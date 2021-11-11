package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	leader "github.com/polygon-io/go-app-ticker-wall/leader"
	tombv2 "gopkg.in/tomb.v2"
)

type ServiceConfig struct {
	Debug        bool
	GRPCPort     int
	HTTPPort     int
	LeaderConfig leader.Config
}

func Run(cfg *ServiceConfig) error {
	// Global top level context.
	tomb, ctx := tombv2.WithContext(context.Background())

	// Start the ticker wall leader.
	clusterLeader, err := leader.New(&cfg.LeaderConfig)
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
