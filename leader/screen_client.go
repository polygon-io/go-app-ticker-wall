package leader

import (
	"errors"
	"sort"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

// UpdateClient is a generic wrapper which is used for all clients which are requesting
// updates be sent to them.
type UpdateClient struct {
	UUID    string
	Screen  *models.Screen
	Updates chan *models.Update
	Stream  models.Leader_JoinClusterServer
}

// CurrentScreenCluster will take the current clients and create a ScreenCluster model.
func (t *Leader) CurrentScreenCluster() *models.ScreenCluster {
	t.RLock()
	defer t.RUnlock()

	res := &models.ScreenCluster{}
	res.Settings = t.PresentationSettings

	for _, client := range t.Clients {
		res.Screens = append(res.Screens, client.Screen)
	}

	return res
}

func (t *Leader) addScreenToCluster(screenClient *UpdateClient) error {
	// Add the client and sort them (asc).
	t.Lock()
	t.Clients = append(t.Clients, screenClient)
	sort.Sort(UpdateClientSlice(t.Clients))
	t.Unlock()

	// Update the cluster
	t.Updates <- &models.Update{
		UpdateType:    int32(models.UpdateTypeCluster),
		ScreenCluster: t.CurrentScreenCluster(),
	}

	return nil
}

func (t *Leader) removeScreenFromCluster(screen *UpdateClient) error {
	t.Lock()

	// Find index of screen.
	screenIndex := -1
	for i, sc := range t.Clients {
		if sc.UUID == screen.UUID {
			screenIndex = i
		}
	}

	// We didn't find this client??
	if screenIndex == -1 {
		t.Unlock()
		return errors.New("unable to find screen when attempting to remove it")
	}

	// Remove the element from the slice.
	t.Clients[screenIndex] = t.Clients[len(t.Clients)-1]
	t.Clients[len(t.Clients)-1] = nil
	t.Clients = t.Clients[:len(t.Clients)-1]

	// Re-sort.
	sort.Sort(UpdateClientSlice(t.Clients))

	t.Unlock()

	// Close the clients updates channel.
	close(screen.Updates)

	logrus.WithFields(logrus.Fields{
		"uuid":   screen.Screen.UUID,
		"index":  screen.Screen.Index,
		"width":  screen.Screen.Width,
		"height": screen.Screen.Height,
	}).Info("Removed screen to cluster.")

	// Update the cluster
	t.Updates <- &models.Update{
		UpdateType:    int32(models.UpdateTypeCluster),
		ScreenCluster: t.CurrentScreenCluster(),
	}

	return nil
}
