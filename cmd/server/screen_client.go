package main

import (
	"errors"
	"sort"

	"github.com/google/uuid"
	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

func (t *TickerWallLeader) addScreenToCluster(screenClient *ScreenClient) error {
	// Generate a unique ID for this client.
	screenClient.UUID = uuid.NewString()

	t.Lock()
	t.ScreenClients = append(t.ScreenClients, screenClient)
	sort.Sort(ScreenClientSlice(t.ScreenClients)) // Sort the screens by their index ( asc ).
	t.Unlock()

	logrus.WithFields(logrus.Fields{
		"index": screenClient.Screen.Index,
		"UUID":  screenClient.UUID,
	}).Info("Screen added to cluster.")

	// Tell all screen clients to update.
	for _, sc := range t.ScreenClients {
		if err := t.queueScreenClientUpdate(sc); err != nil {
			logrus.WithError(err).Error("Unable to send update to screen client.")
			return err
		}
	}

	return nil
}

func (t *TickerWallLeader) removeScreenFromCluster(screen *ScreenClient) error {
	t.Lock()

	// find index of screen..
	screenIndex := -1
	for i, sc := range t.ScreenClients {
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
	t.ScreenClients[screenIndex] = t.ScreenClients[len(t.ScreenClients)-1]
	t.ScreenClients[len(t.ScreenClients)-1] = nil
	t.ScreenClients = t.ScreenClients[:len(t.ScreenClients)-1]

	// Re-sort.
	sort.Sort(ScreenClientSlice(t.ScreenClients))

	t.Unlock()

	// Close the clients updates channel.
	close(screen.Updates)

	logrus.WithFields(logrus.Fields{
		"index": screen.Screen.Index,
		"UUID":  screen.UUID,
	}).Info("Screen removed from cluster.")

	// Tell all screen clients to update.
	for _, sc := range t.ScreenClients {
		if err := t.queueScreenClientUpdate(sc); err != nil {
			logrus.WithError(err).Error("Unable to send update to screen client.")
			return err
		}
	}

	return nil
}

// queueScreenClientUpdate sends an individual screen client the current screen cluster information.
func (t *TickerWallLeader) queueScreenClientUpdate(screenClient *ScreenClient) error {
	t.RLock()
	defer t.RUnlock()

	cluster := &models.ScreenCluster{}
	cluster.NumberOfScreens = int32(len(t.ScreenClients))
	for _, sc := range t.ScreenClients {
		cluster.GlobalViewportSize += int64(sc.Screen.Width)
		cluster.TickerBoxWidth = int32(t.cfg.TickerBoxWidthPx)
		cluster.ScrollSpeed = int32(t.cfg.ScrollSpeed)
		cluster.Screens = append(cluster.Screens, sc.Screen)
	}

	// Create the update container msg.
	update := &models.Update{
		UpdateType:    models.UpdateTypeScreenCluster,
		ScreenCluster: cluster,
	}

	// Put on the queue for this client.
	screenClient.Updates <- update

	return nil
}
