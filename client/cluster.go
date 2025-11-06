package client

import (
	"github.com/massive-com/go-app-ticker-wall/v2/models"
	"github.com/sirupsen/logrus"
)

func (t *ClusterClient) updateScreenCluster(cluster *models.ScreenCluster) {
	logrus.Debug("Updating screen cluster information..")
	t.Lock()
	defer t.Unlock()

	t.Cluster = cluster
}
