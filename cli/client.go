package cli

import (
	"fmt"
	"time"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type ServerClient struct {
	Leader string
	conn   *grpc.ClientConn
	client models.LeaderClient
}

func NewServerClient(leader string) (*ServerClient, error) {
	obj := &ServerClient{
		Leader: leader,
	}
	if err := obj.setup(); err != nil {
		return nil, err
	}
	return obj, nil
}

func (s *ServerClient) setup() error {
	if err := s.startGRPCClient(); err != nil {
		return fmt.Errorf("unable to create grpc connection: %w", err)
	}
	return nil
}

// startGRPCClient creates a new GRPC client connection.
const maxMessageSize = 1024 * 1024 * 1 // 1MB
func (s *ServerClient) startGRPCClient() error {
	logrus.Debug("Connect to gRPC Leader.")

	var opts []grpc.DialOption
	opts = append(opts,
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMessageSize)),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)

	conn, err := grpc.Dial(s.Leader, opts...)
	if err != nil {
		return fmt.Errorf("not able to connect to grpc ticker wall leader: %w", err)
	}

	logrus.Debug("Connected GRPC to Leader.")

	// Set our attributes.
	s.conn = conn
	s.client = models.NewLeaderClient(s.conn)

	logrus.Debug("Created new gRPC client to Leader.")

	return nil
}
