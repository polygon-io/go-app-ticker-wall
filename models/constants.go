package models

// UpdateType is a constant used to define the contents of an update.
type UpdateType int32

const (
	// UpdateTypeUnknown is not a known update type.
	UpdateTypeUnknown UpdateType = 0
	// UpdateTypeCluster updates information about the screen cluster.
	UpdateTypeCluster UpdateType = 1
	// UpdateTypeTickerAdded adds a new ticker to our list.
	UpdateTypeTickerAdded UpdateType = 2
	// UpdateTypeTickerRemoved a ticker has been removed from the list.
	UpdateTypeTickerRemoved UpdateType = 3
	// UpdateTypeTickerUpdate a ticker has been removed from the list.
	UpdateTypeTickerUpdate UpdateType = 4
	// UpdateTypeAnnouncement an announcement has been created.
	UpdateTypeAnnouncement UpdateType = 5
	// UpdateTypePrice means a tickers price has been updated.
	UpdateTypePrice UpdateType = 6
)

// AnnouncementType is used to signify the type of announcement / alert. Different announcement types behave differently.
type AnnouncementType int32

const (
	// AnnouncementTypeInfo is a normal announcement.
	AnnouncementTypeInfo AnnouncementType = 0
	// AnnouncementTypeDanger is an announcement with 'Danger' colors.
	AnnouncementTypeDanger AnnouncementType = 1
	// AnnouncementTypeSuccess is an announcement with 'Success' colors.
	AnnouncementTypeSuccess AnnouncementType = 2
)

// AnnouncementAnimation are the different animation options available for an announcement.
type AnnouncementAnimation int32

const (
	// AnnouncementAnimationElastic uses the Elastic animation pattern.
	AnnouncementAnimationElastic AnnouncementAnimation = 0
	// AnnouncementAnimationBounce uses the Bounce animation pattern.
	AnnouncementAnimationBounce AnnouncementAnimation = 1
	// AnnouncementAnimationEase uses the Easing animation pattern.
	AnnouncementAnimationEase AnnouncementAnimation = 2
	// AnnouncementAnimationBack uses the Back animation pattern.
	AnnouncementAnimationBack AnnouncementAnimation = 3
)
