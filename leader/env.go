package leader

// Config handles the default settings, as well as data client auth.
type Config struct {
	TickerList string
	APIKey     string

	// Presentation Default Settings
	Presentation struct {
		TickerBoxWidthPx  int
		ScrollSpeed       int
		AnimationDuration int
		ShowLogos         bool

		// Data Updates.
		PerTickUpdates bool

		// Color defaults.
		UpColor          map[string]int32
		DownColor        map[string]int32
		FontColor        map[string]int32
		TickerBoxBGColor map[string]int32
		BGColor          map[string]int32
	}
}
