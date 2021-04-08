package main

import (
	"fmt"
	"log"
	"net"

	"github.com/kelseyhightower/envconfig"
	"github.com/polygon-io/go-app-ticker-wall/models"
	"google.golang.org/grpc"
)

var cfg ServiceConfig

type ServiceConfig struct {
	// Service details
	LogLevel    string `split_words:"true" default:"DEBUG"`
	GRPCPort    string `split_words:"true" default:":6886"`
	Location    string `split_words:"true" default:"America/New_York"`
	FileBaseDir string `split_words:"true" default:"./data/node1/"`
	TickerList  string `split_words:"true" default:"AAPL,AMD,NVDA,MSFT,NFLX,LPL,AMZN,SNAP,GME"`
}

func main() {

	// Parse Env Vars:
	err := envconfig.Process("POLY", &cfg)
	if err != nil {
		panic(err.Error())
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 6886))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	tickerWall := NewTickerWallLeader()
	models.RegisterTickerWallLeaderServer(grpcServer, tickerWall)
	grpcServer.Serve(lis)
}
