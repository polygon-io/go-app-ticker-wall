package main

import (
	"context"
	"fmt"

	"github.com/polygon-io/go-app-ticker-wall/client"
	"github.com/sirupsen/logrus"

	"github.com/kelseyhightower/envconfig"
	tombv2 "gopkg.in/tomb.v2"
)

type Config struct {
	// Service details
	LogLevel string `split_words:"true" default:"DEBUG"`
}

func run() error {
	// Global top level context.
	tomb, ctx := tombv2.WithContext(context.Background())

	// Parse Env Vars:
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return err
	}

	// Set Log Levels
	l, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logrus.WithField("err", err).Warn("parse log level")
	} else {
		logrus.SetLevel(l)
	}

	// Ticker wall client.
	tickerWallClient, err := client.New()
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

func main() {
	if err := run(); err != nil {
		logrus.WithError(err).Error("Program exiting")
	}
}
