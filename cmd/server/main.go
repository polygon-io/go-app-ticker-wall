package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	leader "github.com/polygon-io/go-app-ticker-wall/leader"
	"github.com/sirupsen/logrus"
	tombv2 "gopkg.in/tomb.v2"
)

type ServiceConfig struct {
	LogLevel string `split_words:"true" default:"DEBUG"`
	GRPCPort int    `split_words:"true" default:"6886"`
	HTTPPort int    `split_words:"true" default:"6887"`
}

func run() error {
	// Global top level context.
	tomb, ctx := tombv2.WithContext(context.Background())

	// Parse Env Vars:
	var cfg ServiceConfig
	err := envconfig.Process("POLY", &cfg)
	if err != nil {
		return err
	}

	// Set Log Levels
	l, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logrus.WithField("err", err).Warn("parse log level")
	} else {
		logrus.SetLevel(l)
	}

	// Start the ticker wall leader.
	clusterLeader, err := leader.New()
	if err != nil {
		return fmt.Errorf("could not create cluster leader: %w", err)
	}

	tomb.Go(func() error {
		return clusterLeader.Run(ctx)
	})

	// Start the GRPC server.
	tomb.Go(func() error {
		return startGRPC(ctx, cfg.GRPCPort, clusterLeader)
	})

	// Start the HTTP admin server.
	tomb.Go(func() error {
		return runHTTPServer(ctx, cfg.HTTPPort, clusterLeader)
	})

	// Wait for OS signals:
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	tomb.Go(func() error {
		select {
		case <-sigs:
			tomb.Kill(nil)
		case <-tomb.Dying():
			// Exit.
		}
		return nil
	})

	return tomb.Wait()
}

func main() {
	if err := run(); err != nil {
		logrus.WithError(err).Error("Program exiting")
	}
	logrus.Info("bye.")
}
