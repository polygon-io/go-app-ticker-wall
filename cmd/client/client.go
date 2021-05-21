package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/polygon-io/go-app-ticker-wall/models"
	tickerManager "github.com/polygon-io/go-app-ticker-wall/ticker_manager"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// TickerWallClient manages the connection / interaction with leader.
type TickerWallClient struct {
	cfg    *ServiceConfig
	conn   *grpc.ClientConn
	client models.TickerWallLeaderClient
	screen *models.Screen

	manager      tickerManager.TickerManager
	announcement *models.Announcement
}

// NewTickerWallClient creates a new ticker wall client.
func NewTickerWallClient(cfg *ServiceConfig, manager tickerManager.TickerManager) *TickerWallClient {
	obj := &TickerWallClient{
		cfg:     cfg,
		manager: manager,
		screen: &models.Screen{
			Width:              int32(cfg.ScreenWidth),
			Height:             int32(cfg.ScreenHeight),
			Index:              int32(cfg.ScreenIndex),
			ScreenGlobalOffset: 0,
		},
	}
	return obj
}

// Run starts all of our go routines / listeners.
func (t *TickerWallClient) Run(ctx context.Context) error {
	updateListener, err := t.client.RegisterAndListenForUpdates(ctx, t.screen)
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
			return err
		}

		if update == nil {
			logrus.Warning("Update message empty...")
			continue
		}

		switch update.UpdateType {
		case models.UpdateTypeCluster:
			t.updateScreenCluster(update)
		case models.UpdateTypeTicker:
			t.updateTicker(update)
		case models.UpdateTypeAnnouncement:
			t.updateAnnouncement(update)
		default:
			logrus.WithField("updateType", update.UpdateType).Warning("Unknown update type message.")
		}
	}
}

func (t *TickerWallClient) updateTicker(update *models.Update) error {
	return t.manager.UpdateTicker(update.Ticker)
}

func (t *TickerWallClient) updateAnnouncement(update *models.Update) error {
	// TODO: figure out when we should remove the announcement once it's lifespan has ended.
	t.announcement = update.Accouncement
	return nil
}

func (t *TickerWallClient) updateScreenCluster(update *models.Update) error {
	logrus.Debug("Updating screen cluster information..")
	var localizedOffset int64
	for _, screen := range update.ScreenCluster.Screens {
		// This is our index offset
		if int(screen.Index) < t.cfg.ScreenIndex {
			localizedOffset += int64(screen.Width)
		}
	}

	// Configure new presentation settings.
	presentationSettings := &tickerManager.PresentationData{
		ScreenGlobalOffset: localizedOffset,
		NumberOfScreens:    len(update.ScreenCluster.Screens),
		GlobalViewportSize: update.ScreenCluster.GlobalViewportSize,
		TickerBoxWidth:     int(update.ScreenCluster.TickerBoxWidth),
		ScreenWidth:        t.cfg.ScreenWidth,
		ScreenHeight:       t.cfg.ScreenHeight,
		ScrollSpeed:        int(update.ScreenCluster.ScrollSpeed),
	}

	logrus.WithFields(logrus.Fields{
		"globalScreenOffset": presentationSettings.ScreenGlobalOffset,
		"globalViewportSize": presentationSettings.GlobalViewportSize,
		"screens":            presentationSettings.NumberOfScreens,
	}).Debug("Presentation Data Updated")

	// Update our presentation settings.
	t.manager.SetPresentationData(presentationSettings)

	return nil
}

func (t *TickerWallClient) LoadTickers(ctx context.Context) error {
	tickers, err := t.client.GetTickers(ctx, t.screen)
	if err != nil {
		return err
	}

	// Add our tickers to the manager.
	for _, ticker := range tickers.Tickers {
		ticker.PriceChangePercentage = 1 - (ticker.Price / ticker.PreviousClosePrice)
		t.manager.AddTicker(*ticker)

		// ensure logos directory is created
		if err := os.MkdirAll("./logos/", 0755); err != nil {
			return fmt.Errorf("make output dir: %w", err)
		}

		// Download company logo
		if err := downloadLogo(ticker); err != nil {
			return err
		}
	}

	return nil
}

// downloadLogo downloads the logo from our predictable S3 endpoint. This is deprecated,
// so we will need to update this soon...
func downloadLogo(ticker *models.Ticker) error {
	logrus.Debug("Downloading logo for: ", ticker.Ticker)
	url := "https://s3.polygon.io/logos/" + strings.ToLower(ticker.Ticker) + "/logo.png"
	response, e := http.Get(url)
	if e != nil {
		log.Fatal(e)
	}
	defer response.Body.Close()

	// Write it to disk
	file, err := os.Create("./logos/" + ticker.Ticker + ".png")
	if err != nil {
		return err
	}
	defer file.Close()

	// Use io.Copy to just dump the response body to the file.
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	logrus.Debug("Done downloading logo for: ", ticker.Ticker)
	return nil
}

// Close cleans up our current grpc connection.
func (t *TickerWallClient) Close() error {
	return t.conn.Close()
}

// CreateGRPCClient creates a new GRPC client connection.
func (t *TickerWallClient) CreateGRPCClient() error {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMessageSize)))

	conn, err := grpc.Dial(cfg.Leader, opts...)
	if err != nil {
		return fmt.Errorf("not able to connect to grpc ticker wall leader: %w", err)
	}

	// Set our attributes.
	t.conn = conn
	t.client = models.NewTickerWallLeaderClient(t.conn)

	return nil
}
