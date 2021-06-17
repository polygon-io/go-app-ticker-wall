package client

import (
	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

func (t *ClusterClient) updateScreenCluster(cluster *models.ScreenCluster) {
	logrus.Debug("Updating screen cluster information..")
	t.Lock()
	defer t.Unlock()

	t.Cluster = cluster
}
