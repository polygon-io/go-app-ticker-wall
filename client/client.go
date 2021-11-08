package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	tombv2 "gopkg.in/tomb.v2"
)

// Client provides the basic endpoints needed to access the state of this client.
type Client interface {
	GetTickers() []*models.Ticker
	GetSettings() *models.PresentationSettings
	GetCluster() *models.ScreenCluster
	GetScreen() *models.Screen
	GetAnnouncement() *models.Announcement
	GetStatus() *Status
}

const maxMessageSize = 1024 * 1024 * 1 // 1MB

// ClusterClient keeps the client in sync with the leader.
type ClusterClient struct {
	sync.RWMutex
	config Config

	conn   *grpc.ClientConn
	client models.LeaderClient

	// State which will be kept in sync.
	Screen       *models.Screen
	Tickers      []*models.Ticker
	Cluster      *models.ScreenCluster
	Announcement *models.Announcement

	Status *Status
}

// New creates a new ticker wall client.
func New(cfg Config) (*ClusterClient, error) {
	obj := &ClusterClient{
		config: cfg,
		Status: &Status{
			GRPCStatus: GRPCStatusDisconnected,
		},
		Screen: &models.Screen{
			UUID:   uuid.NewString(),
			Width:  int32(cfg.ScreenWidth),
			Height: int32(cfg.ScreenHeight),
			Index:  int32(cfg.ScreenIndex),
		},
	}

	return obj, nil
}

// Run starts all of our go routines / listeners.
func (t *ClusterClient) Run(ctx context.Context) error {
	// Create gRPC connection, close when done.
	for {
		// Check the context incase we cannot ever connect to leader.
		if err := ctx.Err(); err != nil {
			return err
		}

		// Continue trying to connect to GRPC until we eventually connect.
		if err := t.startGRPCClient(); err != nil {
			logrus.Error("Could not create GRPC client to leader.")
			continue
		}

		// We connected, exit loop.
		break
	}
	defer t.Close()

	// Create new tomb for this process.
	tomb, ctx := tombv2.WithContext(ctx)

	// Join the leaders screen cluster, wait for updates.
	tomb.Go(func() error {
		return t.joinCluster(ctx)
	})

	return tomb.Wait()
}

func (t *ClusterClient) joinCluster(ctx context.Context) error {
	// Load in all ticker details.
	if err := t.LoadTickers(ctx); err != nil {
		return err
	}

	// Join cluster, get read stream ( updateListener ) of events.
	updateListener, err := t.client.JoinCluster(ctx, t.Screen)
	if err != nil {
		return err
	}

	// Read loop.
	for {
		// Read message.
		update, err := updateListener.Recv()
		if err != nil {
			logrus.WithError(err).Error("grpc client ending..")

			t.Status.GRPCStatus = GRPCStatusReconnecting
			if err := t.startGRPCClient(); err != nil {
				logrus.WithError(err).Error("grpc - could not reconnect... will continue trying...")
				continue
			}

			// Now that we are reconnected, start over.
			return t.joinCluster(ctx)
		}

		t.Status.GRPCStatus = GRPCStatusConnected

		if update == nil {
			logrus.Warning("Update message empty...")
			continue
		}

		if err := t.processUpdate(update); err != nil {
			return err
		}
	}
}

func (t *ClusterClient) processUpdate(update *models.Update) error {
	var err error
	switch models.UpdateType(update.UpdateType) {
	// Screen cluster has changed.
	case models.UpdateTypeCluster:
		t.updateScreenCluster(update.ScreenCluster)

	// Ticker added.
	case models.UpdateTypeTickerAdded:
		err = t.tickerAdded(update.Ticker)

	// Ticker removed.
	case models.UpdateTypeTickerRemoved:
		err = t.tickerRemoved(update.Ticker)

	// Ticker updated.
	case models.UpdateTypeTickerUpdate:
		// We can again use the tickerAdded method since we dedupe and replace.
		err = t.tickerAdded(update.Ticker)

	// Price of a ticker updated.
	case models.UpdateTypePrice:
		err = t.tickerPriceUpdate(update.PriceUpdate)

	// We have a new announcement.
	case models.UpdateTypeAnnouncement:
		err = t.updateAnnouncement(update.Announcement)

	default:
		logrus.WithField("updateType", update.UpdateType).Warning("Unknown update type message.")
	}

	if err != nil {
		return err
	}

	return nil
}

// Close cleans up our current grpc connection.
func (t *ClusterClient) Close() error {
	return t.conn.Close()
}

// startGRPCClient creates a new GRPC client connection.
func (t *ClusterClient) startGRPCClient() error {
	logrus.Debug("Connect to gRPC Leader.")

	var opts []grpc.DialOption
	opts = append(opts,
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMessageSize)),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)

	conn, err := grpc.Dial(t.config.Leader, opts...)
	if err != nil {
		return fmt.Errorf("not able to connect to grpc ticker wall leader: %w", err)
	}

	logrus.Debug("Connected TCP to Leader.")

	// Set our attributes.
	t.conn = conn
	t.client = models.NewLeaderClient(t.conn)

	logrus.Debug("Created new gRPC client to Leader.")

	return nil
}

// nolint:unparam // This will become more complex in later PR.
func (t *ClusterClient) updateAnnouncement(announcement *models.Announcement) error {
	// TODO: figure out when we should remove the announcement once it's lifespan has ended.
	t.Announcement = announcement
	return nil
}
