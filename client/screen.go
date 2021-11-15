package client

import (
	"context"

	"github.com/sirupsen/logrus"
)

func (t *ClusterClient) UpdateScreen(width, height int) {
	logrus.WithFields(logrus.Fields{
		"widht":  width,
		"height": height,
	}).Debug("Updating screen attributes..")
	t.Lock()
	t.Screen.Height = int32(height)
	t.Screen.Width = int32(width)
	t.Unlock()

	// Let the cluster know about our changes.
	t.broadcastScreenUpdate()
}

func (t *ClusterClient) broadcastScreenUpdate() {
	t.client.UpdateScreen(context.Background(), t.Screen)
}
