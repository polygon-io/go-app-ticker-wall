package main

import (
	"context"
	"fmt"
	"net"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// startGRPC starts the gRPC server. When the given context ends, it will shutdown the gRPC server.
func startGRPC(ctx context.Context, port int, tickerWallLeader models.LeaderServer) error {
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	go func() {
		<-ctx.Done()
		logrus.Debug("Closing gRPC server.")
		grpcServer.Stop()
	}()

	models.RegisterLeaderServer(grpcServer, tickerWallLeader)
	return grpcServer.Serve(lis)
}
