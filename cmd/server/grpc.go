package main

import (
	"context"
	"fmt"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

func (t *TickerWallLeader) RegisterAndListenForUpdates(screen *models.Screen, stream models.TickerWallLeader_RegisterAndListenForUpdatesServer) error {
	logrus.Info("Got Screen: ", screen.Index)
	screenClient := &ScreenClient{
		Screen:  screen,
		Stream:  stream,
		Updates: make(chan *models.Update, 10), // dont block.
	}

	// Add new screen client.
	if err := t.addScreenToCluster(screenClient); err != nil {
		return fmt.Errorf("unable to add new screen client: %w", err)
	}

	// Remove this screen when we close the request.
	defer func() {
		if err := t.removeScreenFromCluster(screenClient); err != nil { // When we disconnect, remove from cluster.
			logrus.WithError(err).Error("Couldn't remove screen..")
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			// Client has disconnected.
			return nil
		case update, ok := <-screenClient.Updates:
			if !ok {
				return nil
			}

			if err := screenClient.Stream.Send(update); err != nil {
				return err
			}
		}
	}
}

// GetTickers returns our current state of ticker data.
func (t *TickerWallLeader) GetTickers(ctx context.Context, screen *models.Screen) (*models.Tickers, error) {
	t.RLock()
	defer t.RUnlock()

	return &models.Tickers{
		Tickers: t.Tickers,
	}, nil
}
