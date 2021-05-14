package main

import (
	"fmt"
	"time"

	"github.com/fogleman/ease"
	"github.com/polygon-io/go-app-ticker-wall/models"
	tickerManager "github.com/polygon-io/go-app-ticker-wall/ticker_manager"
	"github.com/polygon-io/nanovgo"
)

func renderTickers(ctx *nanovgo.Context, mgr tickerManager.TickerManager, globalOffset int64) {
	tickers := mgr.DetermineTickersForRender(globalOffset)
	for _, ticker := range tickers {
		renderTicker(ctx, mgr, ticker, globalOffset)
	}
}

func renderSpecialMessage(ctx *nanovgo.Context, mgr tickerManager.TickerManager, globalOffset int64, announcement *models.Announcement) {
	t := int(time.Now().UnixNano() / int64(time.Millisecond))

	// We are outside of this messages lifespan.
	if t < int(announcement.ShowAtTimestamp) || t > (int(announcement.ShowAtTimestamp)+int(announcement.LifespanMS)+AnimationDuration) {
		return
	}

	screenHeight := mgr.GetPresentationData().ScreenHeight

	// Text Settings.
	textTopStart := float64(-300)
	textTopEnd := float64(140)
	textTop := textTopEnd

	// BG Settings.
	bgBottomStart := float64(0)
	bgBottomEnd := float64(screenHeight)
	bgBottom := bgBottomEnd
	bgTop := (bgBottom - float64(screenHeight))

	if t-int(announcement.ShowAtTimestamp) < AnimationDuration { // Enter animation is in progress.
		diff := t - int(announcement.ShowAtTimestamp)
		percentageCompleted := float64(diff) / float64(AnimationDuration)

		// bg calcs
		bgBottom = bgBottomStart - ((bgBottomStart - bgBottomEnd) * ease.OutElastic(percentageCompleted))
		bgTop = (bgBottom - float64(screenHeight))

		// text calcs
		textTop = textTopStart - ((textTopStart - textTopEnd) * ease.OutElastic(percentageCompleted))

	} else if t > int(announcement.ShowAtTimestamp+announcement.LifespanMS) { // Exit animation in progress.
		diff := t - int(announcement.ShowAtTimestamp+announcement.LifespanMS)
		percentageCompleted := float64(diff) / float64(AnimationDuration)

		// bg calcs
		bgBottom = bgBottomEnd - ((bgBottomEnd - bgBottomStart) * ease.InElastic(percentageCompleted))
		bgTop = (bgBottom - float64(screenHeight))

		// text calcs
		textTop = textTopEnd - ((textTopEnd - textTopStart) * ease.InElastic(percentageCompleted))
	}

	ctx.Save()
	defer ctx.Restore()

	ctx.BeginPath()
	// Determine where the box should start ( may not be on our screen ).
	left := -float32(mgr.GetPresentationData().ScreenGlobalOffset)
	// Position bg.
	ctx.RoundedRect(left, float32(bgTop), float32(mgr.GetPresentationData().GlobalViewportSize), float32(bgBottom), 0)

	// Determine background color based on announcement type:].
	if announcement.AnnouncementType == "danger" {
		ctx.SetFillColor(nanovgo.RGBA(255, 122, 122, 222))
	} else if announcement.AnnouncementType == "success" {
		ctx.SetFillColor(nanovgo.RGBA(122, 255, 122, 222))
	} else {
		ctx.SetFillColor(nanovgo.RGBA(122, 122, 255, 222))
	}

	ctx.Fill()

	ctx.SetFontSize(96.0)
	ctx.SetFontFace("sans-bold")
	ctx.SetTextAlign(nanovgo.AlignCenter | nanovgo.AlignMiddle)

	// ctx.SetFontBlur(0)
	ctx.SetFillColor(nanovgo.RGBA(255, 255, 255, 255))
	middle := (float32(mgr.GetPresentationData().GlobalViewportSize) / 2) - float32(mgr.GetPresentationData().ScreenGlobalOffset)
	ctx.Text(middle, float32(textTop), announcement.Message)
}

func renderTicker(ctx *nanovgo.Context, mgr tickerManager.TickerManager, ticker *tickerManager.Ticker, globalOffset int64) {
	ticker.RLock()
	defer ticker.RUnlock()

	ctx.SetFontFace("sans-bold")
	ctx.SetTextAlign(nanovgo.AlignLeft | nanovgo.AlignTop)
	ctx.SetTextLineHeight(1.2)
	ctx.SetFontSize(156.0)

	// Green or red.
	if ticker.PriceChangePercentage > 0 {
		ctx.SetFillColor(nanovgo.RGBA(51, 255, 51, 255))
	} else {
		ctx.SetFillColor(nanovgo.RGBA(255, 51, 51, 255))
	}

	tickerOffset := mgr.TickerOffset(globalOffset, ticker)

	ctx.TextBox(float32(tickerOffset), 40, 900, ticker.Ticker.Ticker+" $"+fmt.Sprintf("%.2f", ticker.Price))
	ctx.SetFontSize(56)
	ctx.SetFontFace("sans-light")
	ctx.TextBox(float32(tickerOffset), 180, 900, ticker.CompanyName)

}
