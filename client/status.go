package client

type Status struct {
	GRPCStatus GRPCStatus
}

type GRPCStatus int

const (
	GRPCStatusConnected    = 1
	GRPCStatusReconnecting = 2
	GRPCStatusDisconnected = 3
)
