package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/imdario/mergo"
	"github.com/polygon-io/go-app-ticker-wall/leader"
	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

func runHTTPServer(ctx context.Context, port int, leaderObj *leader.Leader) error {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// Register routes.
	r.GET("/v1/cluster", getCluster(leaderObj))
	r.POST("/v1/presentation", updatePresentation(leaderObj))
	r.POST("/v1/announcement", createAnnouncement(leaderObj))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}

	// Gracefully shutdown the HTTP server when context is closed.
	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(ctx); err != nil {
			logrus.WithError(err).Error("Could not shutdown http server.")
		}
	}()

	logrus.Info("HTTP Server Listening on: ", port)
	return srv.ListenAndServe()
}

func createAnnouncement(leaderObj *leader.Leader) func(*gin.Context) {
	return func(c *gin.Context) {
		var announcement *models.Announcement
		if err := c.ShouldBindJSON(&announcement); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Set the display time of the announcement to +100 ms from now.
		startTimer := time.Now().Add(100 * time.Millisecond)
		announcement.ShowAtTimestampMS = startTimer.UnixNano() / int64(time.Millisecond)

		// Tell all screen clients to update.
		leaderObj.Updates <- &models.Update{
			UpdateType:   int32(models.UpdateTypeAnnouncement),
			Announcement: announcement,
		}

		c.JSON(200, gin.H{
			"done":    true,
			"results": announcement,
		})
	}
}

func updatePresentation(leaderObj *leader.Leader) func(*gin.Context) {
	return func(c *gin.Context) {
		// Parse incoming settings.
		var presentationSettings *models.PresentationSettings
		if err := c.ShouldBindJSON(&presentationSettings); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		logrus.Info("Presentation Settings: ", presentationSettings)

		leaderObj.Lock()

		// Merge the new settings into the current settings. This make is so that updating a presentation setting
		// doesn't require all settings, you can just update 1 attribute.
		if err := mergo.MergeWithOverwrite(leaderObj.PresentationSettings, presentationSettings); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		leaderObj.Unlock()

		// Update the cluster
		leaderObj.Updates <- &models.Update{
			UpdateType:    int32(models.UpdateTypeCluster),
			ScreenCluster: leaderObj.CurrentScreenCluster(),
		}

		c.JSON(200, gin.H{
			"done": true,
		})
	}
}

func getCluster(leaderObj *leader.Leader) func(*gin.Context) {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"cluster": leaderObj.CurrentScreenCluster(),
		})
	}
}
