package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	tombv2 "gopkg.in/tomb.v2"
)

var cfg ServiceConfig

type ServiceConfig struct {
	// Service details
	LogLevel   string `split_words:"true" default:"DEBUG"`
	GRPCPort   string `split_words:"true" default:":6886"`
	TickerList string `split_words:"true" default:"AAPL,AMD,NVDA,MSFT,NFLX,LPL,AMZN,SNAP,GME"`
	APIKey     string `split_words:"true" required:"true"` // polygon.io API key.

	// Presentation Settings
	TickerBoxWidthPx  int `split_words:"true" default:"1000"`
	ScrollSpeed       int `split_words:"true" default:"16"`
	AnimationDuration int `split_words:"true" default:"500"`
}

func run() error {
	// Global top level context.
	tomb, ctx := tombv2.WithContext(context.Background())

	// Parse Env Vars:
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

	tickerWall := NewTickerWallLeader(&cfg)
	// Start the ticker wall leader.
	tomb.Go(func() error {
		return tickerWall.Run(ctx)
	})

	// Start the GRPC server.
	tomb.Go(func() error {
		return startGRPC(ctx, &cfg, tickerWall)
	})

	// Wait for OS signals:
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	tomb.Go(func() error {
		select {
		case <-sigs:
			tomb.Kill(nil)
		case <-tomb.Dying():

		}
		logrus.Debug("Tomb dying")
		return nil
	})

	return tomb.Wait()
}

func main() {
	if err := run(); err != nil {
		logrus.WithError(err).Error("Program exiting")
	}
}

func startGRPC(ctx context.Context, cfg *ServiceConfig, tickerWallLeader models.TickerWallLeaderServer) error {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 6886))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	go func() {
		<-ctx.Done()
		logrus.Debug("Closing gRPC server.")
		grpcServer.Stop()
	}()

	models.RegisterTickerWallLeaderServer(grpcServer, tickerWallLeader)
	return grpcServer.Serve(lis)
}
