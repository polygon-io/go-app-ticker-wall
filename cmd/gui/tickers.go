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
	graphSize       = 180
	miniLogoSize    = 64
	miniLogoPadding = 10
	paddingSize     = 20
)

func (g *GUI) renderTicker(ticker *models.Ticker, globalOffset float32) error {
	// Get necessary parameters.
	settings := g.client.GetSettings()

	g.nanoCtx.SetFontFace("sans-bold")
	g.nanoCtx.SetTextAlign(nanovgo.AlignLeft | nanovgo.AlignTop)
	g.nanoCtx.SetTextLineHeight(1.2)
	g.nanoCtx.SetFontSize(156.0)

	// Green or red.
	var rgbaColor nanovgo.Color
	rgbaColor = settings.UpColor.ToNanov()
	if ticker.PriceChangePercentage < 0 {
		rgbaColor = settings.DownColor.ToNanov()
	}

	// Set this tickers font color
	g.nanoCtx.SetFillColor(rgbaColor)

	tickerOffset := g.TickerOffset(globalOffset, ticker)

	// Calculate all the sub item offsets.
	mainTextOffset := tickerOffset + graphSize + (paddingSize * 2)
	subTextOffset := tickerOffset + graphSize + (paddingSize * 2)
	if settings.ShowLogos {
		subTextOffset += miniLogoSize + paddingSize
	}

	// Main text content.
	g.nanoCtx.TextBox(mainTextOffset, 30, 900, ticker.Ticker+" $"+fmt.Sprintf("%.2f", ticker.Price))

	// Sub text.
	diff := ticker.PreviousClosePrice - ticker.Price
	g.nanoCtx.SetFontSize(56)
	g.nanoCtx.SetFontFace("sans-light")
	g.nanoCtx.TextBox(subTextOffset, 170, 900, ticker.CompanyName)
	g.nanoCtx.SetFontSize(32)
	g.nanoCtx.TextBox(subTextOffset, 220, 900, fmt.Sprintf("%+.2f (%+.2f%%)", diff, ticker.PriceChangePercentage))

	g.renderGraph(tickerOffset, 63, graphSize)

	// Render the logo if enabled.
	if settings.ShowLogos {
		g.renderTickerLogo(mainTextOffset+2, miniLogoSize, ticker)
	}

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
