package leader

import (
	"context"
	"fmt"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

func (t *Leader) JoinCluster(screen *models.Screen, stream models.Leader_JoinClusterServer) error {
	logrus.WithFields(logrus.Fields{
		"uuid":   screen.UUID,
		"index":  screen.Index,
		"width":  screen.Width,
		"height": screen.Height,
	}).Info("Adding screen to cluster.")

	// Create update client
	client := &UpdateClient{
		Screen:  screen,
		Stream:  stream,
		Updates: make(chan *models.Update, 100), // dont block.
	}

	// Add new screen client.
	if err := t.addScreenToCluster(client); err != nil {
		return fmt.Errorf("unable to add new screen client: %w", err)
	}

	logrus.Debug("Screen added")

	// Remove this screen when we close the request.
	defer func() {
		if err := t.removeScreenFromCluster(client); err != nil { // When we disconnect, remove from cluster.
			logrus.WithError(err).Error("Couldn't remove screen..")
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			// Client has disconnected.
			logrus.WithField("client", client.Screen.UUID).Debug("Client has disconnected.")
			return nil
		case update, ok := <-client.Updates:
			if !ok {
				return nil
			}

			// logrus.WithField("client", client.Screen.UUID).Debug("Sending Client Update")
			if err := client.Stream.Send(update); err != nil {
				return err
			}
		}
	}
}

// GetScreenCluster returns our current screen cluster.
func (t *Leader) GetScreenCluster(ctx context.Context, empty *models.Empty) (*models.ScreenCluster, error) {
	return t.CurrentScreenCluster(), nil
}

// GetTickers returns our current state of ticker data.
func (t *Leader) GetTickers(ctx context.Context, empty *models.Empty) (*models.Tickers, error) {
	t.RLock()
	defer t.RUnlock()

	return &models.Tickers{
		Tickers: t.Tickers,
	}, nil
}
