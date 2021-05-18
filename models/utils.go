package models

// UpdateType is a constant used to define the contents of an update.
type UpdateType int32

const (
	UpdateTypeUnknown      = -1
	UpdateTypeCluster      = 1
	UpdateTypeTicker       = 2
	UpdateTypeAnnouncement = 3
)
