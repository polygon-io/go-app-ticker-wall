package main

import (
	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

// TickerWallLeader manages this leaders state.
type TickerWallLeader struct {
	// Our view of the entire screen cluster.
	Screens []*models.Screen
}

// NewTickerWallLeader creates a new ticker wall leader.
func NewTickerWallLeader() *TickerWallLeader {
	return &TickerWallLeader{}
}

func (t *TickerWallLeader) RegisterAndListenForUpdates(screen *models.Screen, stream models.TickerWallLeader_RegisterAndListenForUpdatesServer) error {
	logrus.Info("Got Screen: ", screen)
	return nil
}
func (t *TickerWallLeader) ListenForTickerUpdates(screen *models.Screen, stream models.TickerWallLeader_ListenForTickerUpdatesServer) error {

	return nil
}
