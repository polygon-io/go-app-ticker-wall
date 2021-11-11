package leader

import (
	"context"
	"time"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

// Announce will broadcast an announcement update to all clients.
func (t *Leader) Announce(ctx context.Context, announcement *models.Announcement) (*models.Announcement, error) {
	logrus.Debug("New Announcement..", announcement)

	// Set the announcement timestamp.
	announcement.ShowAtTimestampMS = time.Now().UnixMilli() + 200

	// Announce to clients.
	t.Updates <- &models.Update{
		UpdateType:   int32(models.UpdateTypeAnnouncement),
		Announcement: announcement,
	}

	return announcement, nil
}
