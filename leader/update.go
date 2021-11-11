package leader

import (
	"context"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

// UpdatePresentationSettings updates the presentation settings of the cluster and sends out an
// update to all clients.
func (t *Leader) UpdatePresentationSettings(ctx context.Context, newSettings *models.PresentationSettings) (*models.PresentationSettings, error) {
	logrus.Debug("Update presentation settings..", newSettings)

	t.Lock()
	t.PresentationSettings = newSettings
	t.Unlock()

	t.Updates <- &models.Update{
		UpdateType:           int32(models.UpdatePresentationSettings),
		PresentationSettings: t.PresentationSettings,
	}

	return t.PresentationSettings, nil
}
