package manager

import "sync"

type Ticker struct {
	sync.RWMutex

	// Presentation data
	Index int

	// Actual market data
	Ticker                string
	Price                 float64
	PriceChangePercentage float64
	CompanyName           string
}
