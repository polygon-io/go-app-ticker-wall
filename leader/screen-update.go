package leader

import (
	"context"
	"errors"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

// UpdatePresentationSettings updates the presentation settings of the cluster and sends out an
// update to all clients.
func (t *Leader) UpdateScreen(ctx context.Context, newScreenSettings *models.Screen) (*models.Screen, error) {
	logrus.WithFields(logrus.Fields{
		"UUID":   newScreenSettings.UUID,
		"width":  newScreenSettings.Width,
		"height": newScreenSettings.Height,
		"index":  newScreenSettings.Index,
	}).Debug("Update presentation settings..")

	didFind := false
	t.Lock()
	for _, client := range t.Clients {
		// Find the screen we want to update
		if client.Screen.UUID == newScreenSettings.UUID {
			client.Screen = newScreenSettings
			didFind = true
			break
		}
	}
	t.Unlock()

	// Couldn't find correct screen to update.
	if !didFind {
		return nil, errors.New("unable to find screen to update with given UUID")
	}

	// Update the cluster
	t.Updates <- &models.Update{
		UpdateType:    int32(models.UpdateTypeCluster),
		ScreenCluster: t.CurrentScreenCluster(),
	}

	return newScreenSettings, nil
}
