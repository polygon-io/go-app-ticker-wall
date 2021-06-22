package main

import (
	"math"

	"github.com/polygon-io/go-app-ticker-wall/models"
)

// TickerOffset determines what the offset should be for this ticker, on this screen.
// TODO: We should calculate each tickers offset and allow for dynamic width tickers. When
// 	there is a 4 letter ticker with a 4 digit price, it's much wider than a 2 letter ticker and 2 digit price
func (g *GUI) TickerOffset(globalOffset float32, ticker *models.Ticker) float32 {
	// Get necessary parameters.
	settings := g.client.GetSettings()
	cluster := g.client.GetCluster()
	screen := g.client.GetScreen()
	tickers := g.client.GetTickers()
	screenGlobalOffset := cluster.ScreenGlobalOffset(screen.UUID)

	tickerBoxWidth := float32(settings.TickerBoxWidth)
	tapeWidth := float32(float32(len(tickers)) * tickerBoxWidth)

	offset := ((float32(ticker.Index) * tickerBoxWidth) - globalOffset) - screenGlobalOffset

	// Too far left, need to wrap it around.
	if offset < 0 {
		if offset < -(float32(tickerBoxWidth)) {
			offset = tapeWidth - float32(math.Abs(float64(offset)))
		}
	}

	return offset
}

// DetermineTickersForRender takes a global offset and returns the ticker indices which are
// within visiable positions ( should be rendered ).
func (g *GUI) DetermineTickersForRender(globalOffset float32) []*models.Ticker {
	// Get necessary parameters.
	settings := g.client.GetSettings()
	cluster := g.client.GetCluster()
	screen := g.client.GetScreen()
	tickers := g.client.GetTickers()

	// This will be used to build a list of visible tickers at this offset.
	var visibleTickers []*models.Ticker

	screenGlobalOffset := cluster.ScreenGlobalOffset(screen.UUID)

	// Global offset does not necessarily ever reset, so we need to get the localized offset.
	localizedOffset := globalOffset + screenGlobalOffset
	// // logrus.Trace("Localized Offset: ", localizedOffset)

	firstIndex := int(math.Floor(float64(localizedOffset) / float64(settings.TickerBoxWidth)))
	lastIndex := int(math.Floor(float64(localizedOffset+float32(g.windowWidth)) / float64(settings.TickerBoxWidth)))

	// eg: -2
	if firstIndex < 0 {
		boundedFirst := int(float64(len(tickers)) - math.Abs(float64(firstIndex)))
		visibleTickers = append(visibleTickers, tickers[boundedFirst:]...)
		// Now we set first index to 0 since we have the overflow items.
		firstIndex = 0
	}

	if firstIndex > len(tickers) {
		firstIndex = 0
	}

	// If our end index is outside of the bounds.
	boundedLastIndex := lastIndex
	if lastIndex+1 > len(tickers) {
		boundedLastIndex = len(tickers) - 1
	}

	// Add our valid section.
	visibleTickers = append(visibleTickers, tickers[firstIndex:boundedLastIndex+1]...)

	// If we have overflow, now add those.
	if lastIndex+1 > len(tickers) {
		boundedLast := lastIndex - len(tickers)
		visibleTickers = append(visibleTickers, tickers[:boundedLast+1]...)
	}

	return visibleTickers
}
