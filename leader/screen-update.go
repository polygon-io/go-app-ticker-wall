package leader

import (
	"context"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

// UpdatePresentationSettings updates the presentation settings of the cluster and sends out an
// update to all clients.
func (t *Leader) UpdateScreen(ctx context.Context, newScreenSettings *models.Screen) (*models.Screen, error) {
	logrus.Debug("Update presentation settings..", newScreenSettings)

	t.Lock()
	for _, client := range t.Clients {
		// Find the screen we want to update
		if client.Screen.UUID == newScreenSettings.UUID {
			client.Screen = newScreenSettings
		}
	}
	t.Unlock()

	// Update the cluster
	t.Updates <- &models.Update{
		UpdateType:    int32(models.UpdateTypeCluster),
		ScreenCluster: t.CurrentScreenCluster(),
	}

	return newScreenSettings, nil
}
