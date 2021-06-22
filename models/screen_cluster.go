package models

// This has helper functions for ScreenCluster model.

// ScreenSlice is sortable by index.
type ScreenSlice []*Screen

func (a ScreenSlice) Len() int           { return len(a) }
func (a ScreenSlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ScreenSlice) Less(i, j int) bool { return a[i].Index < a[j].Index }

// ScreenGlobalOffset gets the global offset for a given screen UUID in the cluster.
func (s *ScreenCluster) ScreenGlobalOffset(screenUUID string) float32 {
	var offset float32
	for _, scr := range s.Screens {
		// This is our screen, do not add our own width.
		if scr.UUID == screenUUID {
			break
		}

		// Otherwise add this screens offset to the global offset.
		offset += float32(scr.Width)
	}
	return offset
}

// NumberOfScreens returns the total number of screen devices in the cluster.
func (s *ScreenCluster) NumberOfScreens() int {
	return len(s.Screens)
}

// GlobalViewportSize gets the entire pixel width of the cluster.
func (s *ScreenCluster) GlobalViewportSize() int {
	globalViewportSize := 0
	for _, scr := range s.Screens {
		globalViewportSize += int(scr.Width)
	}
	return globalViewportSize
}
