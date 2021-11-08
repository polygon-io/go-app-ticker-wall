package gui

import (
	"context"
	"fmt"

	"github.com/polygon-io/go-app-ticker-wall/client"
	"github.com/sirupsen/logrus"

	tombv2 "gopkg.in/tomb.v2"
)

type Config struct {
	Debug        bool
	ClientConfig client.Config
}

func Run(cfg *Config) error {
	// Global top level context.
	tomb, ctx := tombv2.WithContext(context.Background())

	// Set Log Levels.
	logLevel := logrus.InfoLevel
	if cfg.Debug {
		logLevel = logrus.DebugLevel
	}
	logrus.SetLevel(logLevel)

	// Ticker wall client.
	tickerWallClient, err := client.New(cfg.ClientConfig)
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}
	defer tickerWallClient.Close()

	// Create a new GUI client ( can only be 1 at a time ).
	gui := NewGUI(tickerWallClient)
	defer gui.Close()

	// Setup our GUI
	if err := gui.Setup(); err != nil {
		return fmt.Errorf("could not start gui: %w", err)
	}

	// tomb will context the context
	tomb.Go(func() error {
		return tickerWallClient.Run(ctx)
	})

	// tomb will context the context
	tomb.Go(func() error {
		return gui.Run(ctx)
	})

	err = gui.RenderLoop(ctx)

	tomb.Kill(err)

	return tomb.Wait()
}
