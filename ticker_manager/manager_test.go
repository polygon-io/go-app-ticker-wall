package manager

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/polygon-io/go-app-ticker-wall/models"
)

/* Layout:

[ 1 ][ 2 ][ 3 ][ 4 ]
  [ screen ]

*/
func TestSimpleFirstScreen(t *testing.T) {
	a := assert.New(t)

	mgr := NewDefaultManager(&PresentationData{
		ScreenGlobalOffset: 0,
		ScreenWidth:        1000,
		ScreenHeight:       100,
		TickerBoxWidth:     500,
		ScreenIndex:        1,
	})

	AddTickersToManager(mgr)
	globalOffset := int64(200)

	// Get the indices of which tickers should be rendered on THIS managers screen.
	tickers := mgr.DetermineTickersForRender(globalOffset)
	a.Len(tickers, 3)

	a.Equal("1", tickers[0].Ticker.Ticker)
	a.EqualValues(-200, mgr.TickerOffset(globalOffset, tickers[0]))

	a.Equal("2", tickers[1].Ticker.Ticker)
	a.EqualValues(300, mgr.TickerOffset(globalOffset, tickers[1]))

	a.Equal("3", tickers[2].Ticker.Ticker)
	a.EqualValues(800, mgr.TickerOffset(globalOffset, tickers[2]))
}

/* Layout:

[ 1 ][ 2 ][ 3 ][ 4 ]
     [ screen ]

*/
func TestSimpleFirstScreen2(t *testing.T) {
	a := assert.New(t)

	mgr := NewDefaultManager(&PresentationData{
		ScreenGlobalOffset: 0,
		ScreenWidth:        1000,
		ScreenHeight:       100,
		TickerBoxWidth:     500,
		ScreenIndex:        1,
	})

	AddTickersToManager(mgr)
	globalOffset := int64(500)

	// Get the indices of which tickers should be rendered on THIS managers screen.
	tickers := mgr.DetermineTickersForRender(globalOffset)
	a.Len(tickers, 3)

	a.Equal("2", tickers[0].Ticker.Ticker)
	a.EqualValues(0, mgr.TickerOffset(globalOffset, tickers[0]))

	a.Equal("3", tickers[1].Ticker.Ticker)
	a.EqualValues(500, mgr.TickerOffset(globalOffset, tickers[1]))

	a.Equal("4", tickers[2].Ticker.Ticker)
	a.EqualValues(1000, mgr.TickerOffset(globalOffset, tickers[2]))
}

/* Layout:

[ 1 ][ 2 ][ 3 ][ 4 ]
            [ screen ]

*/
func TestSimpleFirstScreenWrappingForward(t *testing.T) {
	a := assert.New(t)

	mgr := NewDefaultManager(&PresentationData{
		ScreenGlobalOffset: 0,
		ScreenWidth:        1000,
		ScreenHeight:       100,
		TickerBoxWidth:     500,
		ScreenIndex:        1,
	})

	AddTickersToManager(mgr)
	globalOffset := int64(1200)

	// Get the indices of which tickers should be rendered on THIS managers screen.
	tickers := mgr.DetermineTickersForRender(globalOffset)
	a.Len(tickers, 3)

	a.Equal("3", tickers[0].Ticker.Ticker)
	a.EqualValues(-200, mgr.TickerOffset(globalOffset, tickers[0]))

	a.Equal("4", tickers[1].Ticker.Ticker)
	a.EqualValues(300, mgr.TickerOffset(globalOffset, tickers[1]))

	a.Equal("1", tickers[2].Ticker.Ticker)
	a.EqualValues(800, mgr.TickerOffset(globalOffset, tickers[2]))
}

/* Layout:

[ 1 ][ 2 ][ 3 ][ 4 ]
                   [ screen ]

*/
func TestSimpleFirstScreenWrappingForward2(t *testing.T) {
	a := assert.New(t)

	mgr := NewDefaultManager(&PresentationData{
		ScreenGlobalOffset: 0,
		ScreenWidth:        1000,
		ScreenHeight:       100,
		TickerBoxWidth:     500,
		ScreenIndex:        1,
	})

	AddTickersToManager(mgr)
	globalOffset := int64(1900)

	// Get the indices of which tickers should be rendered on THIS managers screen.
	tickers := mgr.DetermineTickersForRender(globalOffset)
	a.Len(tickers, 3)

	a.Equal("4", tickers[0].Ticker.Ticker)
	a.EqualValues(-400, mgr.TickerOffset(globalOffset, tickers[0]))

	a.Equal("1", tickers[1].Ticker.Ticker)
	a.EqualValues(100, mgr.TickerOffset(globalOffset, tickers[1]))

	a.Equal("2", tickers[2].Ticker.Ticker)
	a.EqualValues(600, mgr.TickerOffset(globalOffset, tickers[2]))
}

/* Layout:

[ 1 ][ 2 ][ 3 ][ 4 ][ 1 ][ 2 ][ 3 ][ 4 ]
  [ xxxxx ][ screen ]

*/
func TestSecondScreenWrappingBackward(t *testing.T) {
	a := assert.New(t)

	mgr := NewDefaultManager(&PresentationData{
		ScreenGlobalOffset: 1000,
		ScreenWidth:        1000,
		ScreenHeight:       100,
		TickerBoxWidth:     500,
		ScreenIndex:        2,
	})

	AddTickersToManager(mgr)
	globalOffset := int64(200)

	// Get the indices of which tickers should be rendered on THIS managers screen.
	tickers := mgr.DetermineTickersForRender(globalOffset)
	a.Len(tickers, 3)

	a.Equal("3", tickers[0].Ticker.Ticker)
	a.EqualValues(-200, mgr.TickerOffset(globalOffset, tickers[0]))

	a.Equal("4", tickers[1].Ticker.Ticker)
	a.EqualValues(300, mgr.TickerOffset(globalOffset, tickers[1]))

	a.Equal("1", tickers[2].Ticker.Ticker)
	a.EqualValues(800, mgr.TickerOffset(globalOffset, tickers[2]))
}

func AddTickersToManager(mgr TickerManager) {
	// Create some tickers.
	mgr.AddTicker(models.Ticker{Ticker: "1", Price: 1.00, PriceChangePercentage: 1.00, CompanyName: "1 Inc"})
	mgr.AddTicker(models.Ticker{Ticker: "2", Price: 2.00, PriceChangePercentage: 2.00, CompanyName: "2 Inc"})
	mgr.AddTicker(models.Ticker{Ticker: "3", Price: 3.00, PriceChangePercentage: 3.00, CompanyName: "3 Inc"})
	mgr.AddTicker(models.Ticker{Ticker: "4", Price: 4.00, PriceChangePercentage: 4.00, CompanyName: "4 Inc"})
}
