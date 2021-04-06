package manager

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	globalOffset := 200

	// Get the indices of which tickers should be rendered on THIS managers screen.
	tickers := mgr.DetermineTickersForRender(globalOffset)
	a.Len(tickers, 3)

	a.Equal("1", tickers[0].Ticker)
	a.Equal(-200, mgr.TickerOffset(globalOffset, tickers[0]))

	a.Equal("2", tickers[1].Ticker)
	a.Equal(300, mgr.TickerOffset(globalOffset, tickers[1]))

	a.Equal("3", tickers[2].Ticker)
	a.Equal(800, mgr.TickerOffset(globalOffset, tickers[2]))
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
	globalOffset := 500

	// Get the indices of which tickers should be rendered on THIS managers screen.
	tickers := mgr.DetermineTickersForRender(globalOffset)
	a.Len(tickers, 3)

	a.Equal("2", tickers[0].Ticker)
	a.Equal(0, mgr.TickerOffset(globalOffset, tickers[0]))

	a.Equal("3", tickers[1].Ticker)
	a.Equal(500, mgr.TickerOffset(globalOffset, tickers[1]))

	a.Equal("4", tickers[2].Ticker)
	a.Equal(1000, mgr.TickerOffset(globalOffset, tickers[2]))
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
	globalOffset := 1200

	// Get the indices of which tickers should be rendered on THIS managers screen.
	tickers := mgr.DetermineTickersForRender(globalOffset)
	a.Len(tickers, 3)

	a.Equal("3", tickers[0].Ticker)
	a.Equal(-200, mgr.TickerOffset(globalOffset, tickers[0]))

	a.Equal("4", tickers[1].Ticker)
	a.Equal(300, mgr.TickerOffset(globalOffset, tickers[1]))

	a.Equal("1", tickers[2].Ticker)
	a.Equal(800, mgr.TickerOffset(globalOffset, tickers[2]))
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
	globalOffset := 1900

	// Get the indices of which tickers should be rendered on THIS managers screen.
	tickers := mgr.DetermineTickersForRender(globalOffset)
	a.Len(tickers, 3)

	a.Equal("4", tickers[0].Ticker)
	a.Equal(-400, mgr.TickerOffset(globalOffset, tickers[0]))

	a.Equal("1", tickers[1].Ticker)
	a.Equal(100, mgr.TickerOffset(globalOffset, tickers[1]))

	a.Equal("2", tickers[2].Ticker)
	a.Equal(600, mgr.TickerOffset(globalOffset, tickers[2]))
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
	globalOffset := 200

	// Get the indices of which tickers should be rendered on THIS managers screen.
	tickers := mgr.DetermineTickersForRender(globalOffset)
	a.Len(tickers, 3)

	a.Equal("3", tickers[0].Ticker)
	a.Equal(-200, mgr.TickerOffset(globalOffset, tickers[0]))

	a.Equal("4", tickers[1].Ticker)
	a.Equal(300, mgr.TickerOffset(globalOffset, tickers[1]))

	a.Equal("1", tickers[2].Ticker)
	a.Equal(800, mgr.TickerOffset(globalOffset, tickers[2]))
}

func AddTickersToManager(mgr TickerManager) {
	// Create some tickers.
	mgr.AddTicker("1", 1.00, 1.00, "1 Inc")
	mgr.AddTicker("2", 2.00, 2.00, "2 Inc")
	mgr.AddTicker("3", 3.00, 3.00, "3 Inc")
	mgr.AddTicker("4", 4.00, 4.00, "4 Inc")
}
