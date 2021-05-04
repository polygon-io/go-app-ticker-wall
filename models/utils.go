package models

// UpdateType is a constant used to define the contents of an update.
type UpdateType int32

const (
	UpdateTypeUnknown       = -1
	UpdateTypeScreenCluster = 1
	UpdateTypeScreenTicker  = 2
)
