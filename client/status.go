package client

type Status struct {
	GRPCStatus GRPCStatus
}

// GRPCStatus defines the current status of the given gRPC connection.
type GRPCStatus int

const (
	// GRPCStatusConnected means the connection is established and currently connected. OK.
	GRPCStatusConnected = 1
	// GRPCStatusReconnecting means the connection has closed, and we are trying to reconnect again.
	GRPCStatusReconnecting = 2
	// GRPCStatusDisconnected means the connection has closed.
	GRPCStatusDisconnected = 3
)
