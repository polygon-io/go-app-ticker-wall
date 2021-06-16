package leader

// Config handles the default settings, as well as data client auth.
type Config struct {
	TickerList string `split_words:"true" default:"AAPL,AMD,NVDA"`
	// TickerList string `split_words:"true" default:"AAPL,AMD,NVDA,FB,NFLX,LPL,AMZN,SNAP,NKE,SBUX,SQ,INTC,IBM"`
	APIKey string `split_words:"true" required:"true"` // polygon.io API key.

	// Presentation Default Settings
	Presentation struct {
		TickerBoxWidthPx  int    `split_words:"true" default:"1300"`
		ScrollSpeed       int    `split_words:"true" default:"16"`
		AnimationDuration int    `split_words:"true" default:"500"`
		UpColor           string `split_words:"true" default:"TBI"`
		DownColor         string `split_words:"true" default:"TBI"`
		BGColor           string `split_words:"true" default:"TBI"`
		ShowLogos         bool   `split_words:"true" default:"true"`
	}
}
