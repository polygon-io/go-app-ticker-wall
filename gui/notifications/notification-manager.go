package notifications

import (
	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/polygon-io/nanovgo"
	"github.com/sirupsen/logrus"
)

type Manager struct {
	Notifications []*Notification

	// These attributes are collected on initialization. If any changes to these attributes happen
	// after creation, they will not be updated.
	settings *models.PresentationSettings
	screen   *models.Screen
	cluster  *models.ScreenCluster
}

func NewManager() *Manager {
	mgr := &Manager{}
	return mgr
}

// UpdateAttributes is used to update the attributes needed during rendering.
// TODO: We should come up with a better way to do this.
func (m *Manager) UpdateAttributes(settings *models.PresentationSettings, cluster *models.ScreenCluster, screen *models.Screen) {
	m.settings = settings
	m.cluster = cluster
	m.screen = screen
}

// RenderLoop loops through our current notifications to see if there are any which we should
// call rendering methods on.
func (m *Manager) RenderLoop(ctx *nanovgo.Context) {
	validCount := 0
	didGC := false
	for _, notification := range m.Notifications {
		if notification.HasCompleted {
			didGC = true
			continue
		}

		// This is setting our valid key to this notification, this allows us to do garbage collection of old
		// notifications without allocating a whole new slice.
		m.Notifications[validCount] = notification
		validCount++

		if notification.ShouldRender() {
			notification.Render(ctx)
		}

	}

	if didGC {
		logrus.Debug("GC'ing Notifications. Removed: ", len(m.Notifications)-validCount)
		// Prevent memory leak by erasing values
		for i := validCount; i < len(m.Notifications); i++ {
			m.Notifications[i] = nil
		}
		m.Notifications = m.Notifications[:validCount]
	}
}

func (m *Manager) AddNotification(announcement *models.Announcement) {
	obj := &Notification{
		mgr:          m,
		announcement: announcement,
		HasCompleted: false,
	}
	obj.setup()

	m.Notifications = append(m.Notifications, obj)
}
