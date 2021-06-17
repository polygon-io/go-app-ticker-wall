package main

import (
	"fmt"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/polygon-io/nanovgo"
)

func (g *GUI) renderTickers(globalOffset int64) error {
	tickers := g.DetermineTickersForRender(globalOffset)
	for _, ticker := range tickers {
		if err := g.renderTicker(ticker, globalOffset); err != nil {
			return fmt.Errorf("could not render ticker: %w", err)
		}
	}

	return nil
}

func (g *GUI) renderTicker(ticker *models.Ticker, globalOffset int64) error {
	// Get necessary parameters.
	settings := g.client.GetSettings()

	g.nanoCtx.SetFontFace("sans-bold")
	g.nanoCtx.SetTextAlign(nanovgo.AlignLeft | nanovgo.AlignTop)
	g.nanoCtx.SetTextLineHeight(1.2)
	g.nanoCtx.SetFontSize(156.0)

	// Green or red.
	var rgbaColor *models.RGBA
	rgbaColor = settings.UpColor
	if ticker.PriceChangePercentage < 0 {
		rgbaColor = settings.DownColor
	}

	// Set this tickers font color
	g.nanoCtx.SetFillColor(nanovgo.RGBA(
		uint8(rgbaColor.Red),
		uint8(rgbaColor.Green),
		uint8(rgbaColor.Blue),
		uint8(rgbaColor.Alpha),
	))

	tickerOffset := g.TickerOffset(globalOffset, ticker)

	g.nanoCtx.TextBox(float32(tickerOffset)+logoSize+(logoSize*logoPaddingPercentage), 30, 900, ticker.Ticker+" $"+fmt.Sprintf("%.2f", ticker.Price))
	g.nanoCtx.SetFontSize(56)
	g.nanoCtx.SetFontFace("sans-light")
	g.nanoCtx.TextBox(float32(tickerOffset)+logoSize+(logoSize*logoPaddingPercentage), 170, 900, ticker.CompanyName)

	diff := ticker.PreviousClosePrice - ticker.Price
	g.nanoCtx.SetFontSize(32)
	g.nanoCtx.TextBox(float32(tickerOffset)+logoSize+(logoSize*logoPaddingPercentage), 220, 900, fmt.Sprintf("%+.2f (%+.2f%%)", diff, ticker.PriceChangePercentage))

	// Paint the logo
	// imgPaint := nanovgo.ImagePattern(float32(tickerOffset), 63, logoSize, logoSize, 0.0/180.0*nanovgo.PI, int(ticker.Ticker.Img), 1)
	// g.nanoCtx.BeginPath()
	// g.nanoCtx.RoundedRect(float32(tickerOffset), 63, logoSize, logoSize, 5)
	// g.nanoCtx.SetFillPaint(imgPaint)
	// g.nanoCtx.Fill()

	return nil
}
