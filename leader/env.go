package leader

// Config handles the default settings, as well as data client auth.
type Config struct {
	TickerList string `split_words:"true" default:"AAPL,AMD,NVDA"`
	// TickerList string `split_words:"true" default:"AAPL,AMD,NVDA,FB,NFLX,LPL,AMZN,SNAP,NKE,SBUX,SQ,INTC,IBM"`
	APIKey string `split_words:"true" required:"true"` // polygon.io API key.

	// Presentation Default Settings
	Presentation struct {
		TickerBoxWidthPx  int              `split_words:"true" default:"1100"`
		ScrollSpeed       int              `split_words:"true" default:"16"`
		AnimationDuration int              `split_words:"true" default:"500"`
		ShowLogos         bool             `split_words:"true" default:"true"`
		UpColor           map[string]int32 `split_words:"true" default:"red:51,green:255,blue:51,alpha:255"`
		DownColor         map[string]int32 `split_words:"true" default:"red:255,green:51,blue:51,alpha:255"`
		BGColor           map[string]int32 `split_words:"true" default:"red:1,green:25,blue:55,alpha:122"`
	}
}
