package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

func (t *TickerWallLeader) runHTTPServer(ctx context.Context) error {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// Register routes.
	r.GET("/v1/screens", t.getScreens)
	// r.POST("/v1/message")
	r.POST("/v1/presentation", t.updatePresentation)

	srv := &http.Server{
		Addr:    cfg.HTTPPort,
		Handler: r,
	}

	// Gracefully shutdown the HTTP server when context is closed.
	go func() {
		<-ctx.Done()
		srv.Shutdown(ctx)
	}()

	logrus.Info("HTTP Server Listening on: ", t.cfg.HTTPPort)
	return srv.ListenAndServe()
}

func (t *TickerWallLeader) updatePresentation(c *gin.Context) {
	// Parse incoming settings.
	var postScreenCluster *models.ScreenCluster
	if err := c.ShouldBindJSON(&postScreenCluster); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logrus.Info("post: ", postScreenCluster)

	hasUpdated := false

	t.Lock()

	// Scroll speed changes.
	if postScreenCluster.ScrollSpeed != t.clusterConfig.ScrollSpeed && postScreenCluster.ScrollSpeed > 0 {
		hasUpdated = true
		t.clusterConfig.ScrollSpeed = postScreenCluster.ScrollSpeed
	}

	// Ticker box width changes.
	if postScreenCluster.TickerBoxWidth != t.clusterConfig.TickerBoxWidth {
		hasUpdated = true
		t.clusterConfig.TickerBoxWidth = postScreenCluster.TickerBoxWidth
	}

	t.Unlock()

	if hasUpdated {
		// Tell all screen clients to update.
		for _, sc := range t.ScreenClients {
			if err := t.queueScreenClientUpdate(sc); err != nil {
				logrus.WithError(err).Error("Unable to send update to screen client.")
				c.JSON(http.StatusBadRequest, gin.H{"error": "unable to update screen clients."})
				return
			}
		}
	}

	c.JSON(200, gin.H{
		"done": true,
	})

}

func (t *TickerWallLeader) getScreens(c *gin.Context) {
	t.RLock()
	defer t.RUnlock()

	var screens []*models.Screen
	for _, screenClient := range t.ScreenClients {
		screens = append(screens, screenClient.Screen)
	}

	c.JSON(200, gin.H{
		"cluster": screens,
	})
}
