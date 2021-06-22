package client

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
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
func New() (*ClusterClient, error) {
	// Parse Env Vars:
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

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
	if err := t.startGRPCClient(); err != nil {
		return err
	}
	defer t.Close()

	// Create new tomb for this process.
	tomb, ctx := tombv2.WithContext(ctx)

	// Join the leaders screen cluster, wait for updates.
	tomb.Go(func() error {
		return t.joinCluster(ctx)
	})

	// Load in all ticker data.
	tomb.Go(func() error {
		return t.LoadTickers(ctx)
	})

	return tomb.Wait()
}

func (t *ClusterClient) joinCluster(ctx context.Context) error {
	updateListener, err := t.client.JoinCluster(ctx, t.Screen)
	if err != nil {
		return err
	}

	for {
		// Read message.
		update, err := updateListener.Recv()
		if err != nil {
			if err == io.EOF {
				logrus.Info("No more messages from leader.")
			}
			logrus.WithError(err).Error("grpc client ending..")
			t.Status.GRPCStatus = GRPCStatusDisconnected
			return err
		}

		t.Status.GRPCStatus = GRPCStatusConnected

		logrus.Info("Got Update: ", update.UpdateType)

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
	switch models.UpdateType(update.UpdateType) {

	// Screen cluster has changed.
	case models.UpdateTypeCluster:
		t.updateScreenCluster(update.ScreenCluster)

	// Ticker added.
	case models.UpdateTypeTickerAdded:
		if err := t.tickerAdded(update.Ticker); err != nil {
			return err
		}

	// Ticker removed.
	case models.UpdateTypeTickerRemoved:
		if err := t.tickerRemoved(update.Ticker); err != nil {
			return err
		}

	// Price of a ticker updated.
	case models.UpdateTypePrice:
		if err := t.tickerPriceUpdate(update.PriceUpdate); err != nil {
			return err
		}

	// We have a new announcement.
	case models.UpdateTypeAnnouncement:
		t.updateAnnouncement(update.Announcement)

	default:
		logrus.WithField("updateType", update.UpdateType).Warning("Unknown update type message.")
	}

	return nil
}

// Close cleans up our current grpc connection.
func (t *ClusterClient) Close() error {
	return t.conn.Close()
}

// startGRPCClient creates a new GRPC client connection.
func (t *ClusterClient) startGRPCClient() error {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMessageSize)))

	conn, err := grpc.Dial(t.config.Leader, opts...)
	if err != nil {
		return fmt.Errorf("not able to connect to grpc ticker wall leader: %w", err)
	}

	// Set our attributes.
	t.conn = conn
	t.client = models.NewLeaderClient(t.conn)

	return nil
}

func (t *ClusterClient) updateAnnouncement(announcement *models.Announcement) error {
	// TODO: figure out when we should remove the announcement once it's lifespan has ended.
	t.Announcement = announcement
	return nil
}
