package main

import (
	"fmt"
	"math"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/polygon-io/nanovgo"
)

func (g *GUI) renderTickers(globalOffset float32) error {
	tickers := g.DetermineTickersForRender(globalOffset)
	for _, ticker := range tickers {
		g.renderTicker(ticker, globalOffset)
	}

	return nil
}

const (
	graphSize               = 180
	graphViewportPercentage = .05 // 5% viewport movement range.

	// Ticker box settings.
	tickerBoxHeight       = 240
	tickerBoxMargin       = 30
	tickerBoxPadding      = 50
	tickerBoxBorderRadius = 8

	// Font sizes.
	upperRowFontSize  = 96
	bottomRowFontSize = 58

	maxCompanyNameCharacters = 14
)

// renderTickerBg sets the background of the ticker box to a solid color.
func (g *GUI) renderTickerBg(leftOffset float32) {
	screen := g.client.GetScreen()
	settings := g.client.GetSettings()

	topOffset := float32((screen.Height / 2) - (tickerBoxHeight / 2))
	leftOffset += (tickerBoxMargin / 2)
	boxWidth := float32(settings.TickerBoxWidth) - tickerBoxMargin

	// Set BG color
	g.nanoCtx.BeginPath()
	g.nanoCtx.RoundedRect(leftOffset, topOffset, boxWidth, tickerBoxHeight, tickerBoxBorderRadius)
	g.nanoCtx.SetFillColor(settings.TickerBoxBGColor.ToNanov())
	g.nanoCtx.Fill()
}

func (g *GUI) renderTicker(ticker *models.Ticker, globalOffset float32) {
	// Get necessary parameters.
	settings := g.client.GetSettings()
	screen := g.client.GetScreen()
	tickerOffset := g.TickerOffset(globalOffset, ticker)

	// Render background rectangle.
	g.renderTickerBg(tickerOffset)

	// Calculate offsets.
	offsetLeft := (tickerOffset + (tickerBoxMargin / 2)) + tickerBoxPadding
	offsetTop := float32((screen.Height / 2) - (tickerBoxHeight / 2))
	offsetRight := ((tickerOffset + float32(settings.TickerBoxWidth)) - tickerBoxMargin) - tickerBoxPadding

	// Calculate the Y offset for the two rows. Using percentages so if we change
	// ticker box size, it should scale accordingly.
	upperRowTopOffset := offsetTop + (tickerBoxHeight * .33)
	lowerRowTopOffset := offsetTop + (tickerBoxHeight * .66)

	// Actual text rendering ---

	// Ticker.
	g.nanoCtx.SetFontFace("sans-bold")
	g.nanoCtx.SetTextAlign(nanovgo.AlignLeft | nanovgo.AlignMiddle)
	g.nanoCtx.SetFontSize(upperRowFontSize)
	g.nanoCtx.SetFillColor(settings.FontColor.ToNanov())
	g.nanoCtx.TextBox(offsetLeft, upperRowTopOffset, 900, ticker.Ticker)

	// Price.
	textString := fmt.Sprintf("%.2f", ticker.Price)
	boundedTextWidth, _ := g.nanoCtx.TextBounds(0, 0, textString)
	g.nanoCtx.Text(offsetRight-boundedTextWidth, upperRowTopOffset, textString)

	// Company Name.
	g.nanoCtx.SetFontSize(bottomRowFontSize)
	g.nanoCtx.SetFontFace("sans-light")
	companyName := ticker.CompanyName
	if len(companyName) >= maxCompanyNameCharacters {
		companyName = companyName[:(maxCompanyNameCharacters-3)] + "..."
	}
	g.nanoCtx.TextBox(offsetLeft, lowerRowTopOffset, 900, companyName)

	// Percentage Gained / Loss test.
	directionalColor := settings.UpColor
	if ticker.PriceChangePercentage < 0 {
		directionalColor = settings.DownColor
	}
	priceDifference := ticker.Price - ticker.PreviousClosePrice
	g.nanoCtx.SetFillColor(directionalColor.ToNanov())
	textString = fmt.Sprintf("%+.2f (%+.2f%%)", priceDifference, ticker.PriceChangePercentage)
	boundedTextWidth, _ = g.nanoCtx.TextBounds(0, 0, textString)
	g.nanoCtx.Text(offsetRight-boundedTextWidth, lowerRowTopOffset, textString)

	// Graph.
	g.renderGraph(ticker, offsetLeft+400, 63, graphSize, directionalColor)
}

func (g *GUI) renderGraph(ticker *models.Ticker, x, y, width float32, color *models.RGBA) {
	g.drawGraph(g.nanoCtx, ticker, x, y, width, width, 2, color)
}

func (g *GUI) drawGraph(ctx *nanovgo.Context, ticker *models.Ticker, x, y, w, h, t float32, color *models.RGBA) {
	points := len(ticker.Aggs)

	// if we have no data, don't continue.
	if points < 1 {
		return
	}

	sx := make([]float32, points)
	sy := make([]float32, points)
	dx := w / float32(points-1)

	// Generate graph points.
	var min, max float64
	for i, agg := range ticker.Aggs {
		// Check if max.
		if agg.Price > max {
			max = agg.Price
		}

		// Check if min.
		if agg.Price < min || min == 0 {
			min = agg.Price
		}

		// Set X,Y for this point.
		sy[i] = float32(agg.Price)
		sx[i] = x + float32(i)*dx
	}

	// Middle of our range.
	midRange := float32((min + max) / 2)

	// Now we must normalize Y axis to fix in our bounds.
	var absMax float32
	for i, val := range sy {
		sy[i] = (val - midRange) / midRange
		absValue := float32(math.Abs(float64(sy[i])))
		if absValue > absMax {
			absMax = absValue
		}
	}

	// If our values are outside of the viewport range percentage, we must squish values
	// to be inside our desired viewport range percentage.
	if absMax > graphViewportPercentage {
		for i, val := range sy {
			sy[i] = (val / absMax) * graphViewportPercentage
		}
	}

	// Change percentage diff to pixel offsets:
	middleOfViewport := h / 2
	baseMultiplier := (middleOfViewport / graphViewportPercentage)
	for i, val := range sy {
		sy[i] = (y + h) - ((baseMultiplier * val) + middleOfViewport)
	}

	ctx.BeginPath()
	ctx.MoveTo(sx[0], sy[0])
	for i := 1; i < points; i++ {
		ctx.LineTo(sx[i], sy[i])
	}
	ctx.SetStrokeColor(color.ToNanov())
	ctx.SetStrokeWidth(4.0)
	ctx.Stroke()

	ctx.BeginPath()
	ctx.Circle(sx[points-1], sy[points-1], 6.0)
	ctx.SetFillColor(color.ToNanov())
	ctx.Fill()

	ctx.SetStrokeWidth(1.0)
}
