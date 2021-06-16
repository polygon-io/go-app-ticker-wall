package main

import (
	"math"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

// TickerOffset determines what the offset should be for this ticker, on this screen.
func (g *GUI) TickerOffset(globalOffset int64, ticker *models.Ticker) int64 {
	g.client.RLock()
	defer g.client.RUnlock()

	tickerBoxWidth := int(g.client.Cluster.Settings.TickerBoxWidth)
	screenGlobalOffset := g.client.Cluster.ScreenGlobalOffset(g.client.Screen.UUID)
	localizedOffset := (globalOffset % int64(len(g.client.Tickers)*tickerBoxWidth))
	offset := (int64(int(ticker.Index)*tickerBoxWidth) - localizedOffset) - int64(screenGlobalOffset)

	// Too far left, need to wrap it around.
	if offset < 0 {
		if offset < -(int64(tickerBoxWidth)) {
			offset = int64(len(g.client.Tickers)*tickerBoxWidth) - int64(math.Abs(float64(offset)))
		}
	}
	return offset
}

// DetermineTickersForRender takes a global offset and returns the ticker indices which are
// within visiable positions ( should be rendered ).
func (g *GUI) DetermineTickersForRender(globalOffset int64) []*models.Ticker {
	// This will be used to build a list of visible tickers at this offset.
	var visibleTickers []*models.Ticker

	presentationSettings := g.client.Cluster.Settings
	tickers := g.client.GetTickers()
	screenGlobalOffset := g.client.Cluster.ScreenGlobalOffset(g.client.Screen.UUID)

	// Global offset does not necessarily ever reset, so we need to get the localized offset.
	localizedOffset := (globalOffset % int64(len(tickers)*int(presentationSettings.TickerBoxWidth))) + int64(screenGlobalOffset)
	// logrus.Trace("Localized Offset: ", localizedOffset)

	firstIndex := int(math.Floor(float64(localizedOffset) / float64(presentationSettings.TickerBoxWidth)))
	lastIndex := int(math.Floor(float64(localizedOffset+int64(g.windowWidth)) / float64(presentationSettings.TickerBoxWidth)))

	logrus.Trace("offsets: ", firstIndex, lastIndex)

	// eg: -2
	if firstIndex < 0 {
		boundedFirst := int(float64(len(tickers)) - math.Abs(float64(firstIndex)))
		logrus.Trace("first index short: ", boundedFirst)
		visibleTickers = append(visibleTickers, tickers[boundedFirst:]...)
		// Now we set first index to 0 since we have the overflow items.
		firstIndex = 0
	}

	if firstIndex > len(tickers) {
		// logrus.Info("Invalid slice bounds - alertttt ")
		firstIndex = 0
	}

	// If our end index is outside of the bounds.
	boundedLastIndex := lastIndex
	if lastIndex+1 > len(tickers) {
		boundedLastIndex = len(tickers) - 1
		logrus.Trace("last index long: ", boundedLastIndex)
	}

	// Add our valid section.
	visibleTickers = append(visibleTickers, tickers[firstIndex:boundedLastIndex+1]...)

	// If we have overflow, now add those.
	if lastIndex+1 > len(tickers) {
		boundedLast := lastIndex - len(tickers)
		logrus.Trace("last index long2: ", boundedLast)
		visibleTickers = append(visibleTickers, tickers[:boundedLast+1]...)
	}

	return visibleTickers
}
