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
		if err := g.renderTicker(ticker, globalOffset); err != nil {
			return fmt.Errorf("could not render ticker: %w", err)
		}
	}

	return nil
}

const (
	graphSize    = 180
	miniLogoSize = 64

	// Ticker box settings
	tickerBoxHeight       = 240
	tickerBoxMargin       = 30
	tickerBoxPadding      = 50
	tickerBoxBorderRadius = 8

	// Font sizes
	upperRowFontSize  = 96
	bottomRowFontSize = 58

	maxCompanyNameCharacters = 14
)

// renderTickerBg sets the background of the ticker box to a solid color.
func (g *GUI) renderTickerBg(leftOffset float32) {
	screen := g.client.GetScreen()
	settings := g.client.GetSettings()

	topOffset := float32((screen.Height / 2) - (tickerBoxHeight / 2))
	leftOffset = leftOffset + (tickerBoxMargin / 2)
	boxWidth := float32(settings.TickerBoxWidth) - tickerBoxMargin

	// Set BG color
	g.nanoCtx.BeginPath()
	g.nanoCtx.RoundedRect(leftOffset, topOffset, boxWidth, tickerBoxHeight, tickerBoxBorderRadius)
	g.nanoCtx.SetFillColor(settings.TickerBoxBGColor.ToNanov())
	g.nanoCtx.Fill()
}

func (g *GUI) renderTicker(ticker *models.Ticker, globalOffset float32) error {
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
	g.nanoCtx.TextBox(offsetLeft, float32(upperRowTopOffset), 900, ticker.Ticker)

	// Company Name.
	g.nanoCtx.SetFontSize(bottomRowFontSize)
	g.nanoCtx.SetFontFace("sans-light")
	companyName := ticker.CompanyName
	if len(companyName) >= maxCompanyNameCharacters {
		companyName = companyName[:(maxCompanyNameCharacters-3)] + "..."
	}
	g.nanoCtx.TextBox(offsetLeft, float32(lowerRowTopOffset), 900, companyName)

	// Price.
	g.nanoCtx.SetFontSize(upperRowFontSize)
	g.nanoCtx.SetFontFace("sans-bold")
	textString := fmt.Sprintf("%.2f", ticker.Price)
	boundedText, _ := g.nanoCtx.TextBounds(0, 0, textString)
	g.nanoCtx.Text(offsetRight-boundedText, upperRowTopOffset, textString)

	// Percentage Gained / Loss test.
	directionalColor := settings.UpColor
	if ticker.PriceChangePercentage < 0 {
		directionalColor = settings.DownColor
	}
	priceDifference := ticker.PreviousClosePrice - ticker.Price
	g.nanoCtx.SetFillColor(directionalColor.ToNanov())
	g.nanoCtx.SetFontSize(bottomRowFontSize)
	g.nanoCtx.SetFontFace("sans-light")
	textString = fmt.Sprintf("%+.2f (%+.2f%%)", priceDifference, ticker.PriceChangePercentage)
	boundedText, _ = g.nanoCtx.TextBounds(0, 0, textString)
	g.nanoCtx.Text(offsetRight-boundedText, lowerRowTopOffset, textString)

	// Graph.
	g.renderGraph(offsetLeft+400, 63, graphSize)

	// Render the logo if enabled.
	// if settings.ShowLogos {
	// 	g.renderTickerLogo(leftTextOffset+2, miniLogoSize, ticker)
	// }

	return nil
}

func (g *GUI) renderTickerLogo(offset, logoSize float32, ticker *models.Ticker) error {
	tickerImg := g.logos.GetTickerImage(ticker)
	if tickerImg == nil {
		return nil
	}

	// Paint the logo
	imgPaint := nanovgo.ImagePattern(offset, 182.5, logoSize, logoSize, 0.0/180.0*nanovgo.PI, int(tickerImg.NanovImgID), 1)
	g.nanoCtx.BeginPath()
	g.nanoCtx.RoundedRect(offset, 182.5, logoSize, logoSize, 5)
	g.nanoCtx.SetFillPaint(imgPaint)
	g.nanoCtx.Fill()

	return nil
}

func (g *GUI) renderGraph(x, y, width float32) {
	g.drawGraph(g.nanoCtx, x, y, width, width, 2)
}

func cosF(a float32) float32 {
	return float32(math.Cos(float64(a)))
}
func sinF(a float32) float32 {
	return float32(math.Sin(float64(a)))
}

func (g *GUI) drawGraph(ctx *nanovgo.Context, x, y, w, h, t float32) {
	settings := g.client.GetSettings()

	// green := settings.UpColor.ToNanov()
	red := settings.DownColor.ToNanov()

	const points = 12
	var sx, sy [points]float32
	dx := w / (points - 1)

	samples := []float32{
		0.1,
		0.2,
		0.3,
		0.4,
		0.3,
		0.4,
		0.45,
		0.32,
		0.299,
		0.2,
		0.15,
		(1 + sinF(t*1.2345+cosF(t*0.33457)*0.44)) * 0.5,
		(1 + sinF(t*0.68363+cosF(t*1.3)*1.55)) * 0.5,
		(1 + sinF(t*1.1642+cosF(t*0.33457)*1.24)) * 0.5,
		(1 + sinF(t*0.56345+cosF(t*1.63)*0.14)) * 0.5,
		(1 + sinF(t*1.6245+cosF(t*0.254)*0.3)) * 0.5,
		(1 + sinF(t*0.345+cosF(t*0.03)*0.6)) * 0.5,
		(1 + sinF(t*1.2345+cosF(t*0.33457)*0.44)) * 0.5,
		(1 + sinF(t*0.68363+cosF(t*1.3)*1.55)) * 0.5,
		(1 + sinF(t*1.1642+cosF(t*0.33457)*1.24)) * 0.5,
		(1 + sinF(t*0.56345+cosF(t*1.63)*0.14)) * 0.5,
		(1 + sinF(t*1.6245+cosF(t*0.254)*0.3)) * 0.5,
		(1 + sinF(t*0.345+cosF(t*0.03)*0.6)) * 0.5,
	}

	for i := 0; i < points; i++ {
		sx[i] = x + float32(i)*dx
		sy[i] = y + h*samples[i]*0.8
	}

	ctx.BeginPath()
	ctx.MoveTo(sx[0], sy[0])
	for i := 1; i < points; i++ {
		ctx.BezierTo(sx[i-1]+dx*0.5, sy[i-1], sx[i]-dx*0.5, sy[i], sx[i], sy[i])
	}
	ctx.SetStrokeColor(red)
	ctx.SetStrokeWidth(2.0)
	ctx.Stroke()

	ctx.BeginPath()
	ctx.Circle(sx[points-1], sy[points-1], 6.0)
	ctx.SetFillColor(red)
	ctx.Fill()

	ctx.SetStrokeWidth(1.0)
}
