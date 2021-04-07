package main

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/fogleman/ease"
	"github.com/goxjs/gl"
	"github.com/goxjs/glfw"
	tickerManager "github.com/polygon-io/go-app-ticker-wall/ticker_manager"
	"github.com/polygon-io/nanovgo"
	"github.com/polygon-io/nanovgo/perfgraph"
	"github.com/sirupsen/logrus"
)

type Pos struct {
	sync.RWMutex
	left float32
}

var (
	ScreenWidth     = 1200
	ScreenHeight    = 300
	NumberOfScreens = 3

	// AnimationDuration is the length of animations.
	AnimationDuration = 750 // ms
)

func main() {
	err := glfw.Init(gl.ContextWatcher)
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	window, err := glfw.CreateWindow(ScreenWidth, ScreenHeight, "Polygon Ticker Wall", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	// ctx, err := nanovgo.NewContext(0)
	ctx, err := nanovgo.NewContext(0)
	defer ctx.Delete()
	if err != nil {
		panic(err)
	}

	glfw.SwapInterval(0)

	ctx.CreateFont("sans", "fonts/Roboto-Regular.ttf")
	ctx.CreateFont("sans-light", "fonts/Roboto-Light.ttf")
	ctx.CreateFont("sans-bold", "fonts/Roboto-Bold.ttf")

	// pos := &Pos{
	// 	left: -1920,
	// }
	// go func() {
	// 	for {
	// 		pos.Lock()
	// 		pos.left += .5
	// 		pos.Unlock()
	// 		time.Sleep(4 * time.Millisecond)
	// 	}
	// }()

	numbPtr := flag.Int("screenindex", 0, "Screen Index")
	flag.Parse()

	// Ticker Manager
	mgr := tickerManager.NewDefaultManager(&tickerManager.PresentationData{
		ScreenWidth:        ScreenWidth,
		ScreenHeight:       ScreenHeight,
		ScreenGlobalOffset: (ScreenWidth * *numbPtr),
		TickerBoxWidth:     1000,
		ScreenIndex:        *numbPtr,
		NumberOfScreens:    NumberOfScreens,
		GlobalViewportSize: (NumberOfScreens * ScreenWidth),
	})

	mgr.AddTicker("AAPL", 355.65, 1.05, "Apple Inc.")
	mgr.AddTicker("AMD", 241.65, -.05, "Advanced Micro Devices Inc.")
	mgr.AddTicker("BRK.B", 955.65, .35, "Berkshire Hathaway Inc.")
	mgr.AddTicker("SNAP", 55.65, 2.19, "Snap Inc.")
	mgr.AddTicker("MSFT", 255.65, -0.19, "Microsoft Inc.")
	mgr.AddTicker("NFLX", 565.65, 4.19, "Netflix Inc.")

	fps := perfgraph.NewPerfGraph("Frame Time", "sans")
	fbWidth, fbHeight := window.GetFramebufferSize()
	winWidth, winHeight := window.GetSize()
	pixelRatio := float32(fbWidth) / float32(winWidth)
	gl.Viewport(0, 0, fbWidth, fbHeight)

	ctx.SetFontFace("sans")
	ctx.SetTextAlign(nanovgo.AlignLeft | nanovgo.AlignTop)
	ctx.SetTextLineHeight(1.2)

	specialMessage := true
	startTimer := time.Now().Add(1 * time.Minute)
	startTimer = startTimer.Truncate(time.Minute)
	specialMessageTimeActivate := startTimer.UnixNano() / int64(time.Millisecond)
	logrus.Info("activation time: ", specialMessageTimeActivate)

	for !window.ShouldClose() {
		fps.UpdateGraph()
		gl.ClearColor(0, 0, 0, 0)
		gl.Clear(gl.COLOR_BUFFER_BIT)
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		gl.Enable(gl.CULL_FACE)
		gl.Disable(gl.DEPTH_TEST)
		ctx.BeginFrame(winWidth, winHeight, pixelRatio)
		// ctx.Save()

		t := int(time.Now().UnixNano() / int64(time.Millisecond*10))
		// println(t)
		// Actual application drawing.
		// pos.RLock()
		renderTickers(ctx, mgr, t)
		// pos.RUnlock()

		if specialMessage {
			renderSpecialMessage(ctx, mgr, t, "Very Important Special Message... Read it!", int(specialMessageTimeActivate), 5000)
		}

		// ctx.Restore()
		fps.RenderGraph(ctx, -50, -50)
		ctx.EndFrame()
		gl.Enable(gl.DEPTH_TEST)
		window.SwapBuffers()
		glfw.PollEvents()
		// time.Sleep(time.Millisecond * 16)
	}

}

func renderTickers(ctx *nanovgo.Context, mgr tickerManager.TickerManager, globalOffset int) {
	tickers := mgr.DetermineTickersForRender(globalOffset)
	for _, ticker := range tickers {
		renderTicker(ctx, mgr, ticker, globalOffset)
	}
}

func renderSpecialMessage(ctx *nanovgo.Context, mgr tickerManager.TickerManager, globalOffset int, message string, activationTime int, visibleLifetimeMS int) {
	t := int(time.Now().UnixNano() / int64(time.Millisecond))

	// We are outside of this messages lifespan.
	if t < activationTime || t > (activationTime+visibleLifetimeMS+AnimationDuration) {
		return
	}

	// Text Settings.
	textTopStart := float64(-300)
	textTopEnd := float64(140)
	textTop := textTopEnd

	// BG Settings.
	bgBottomStart := float64(0)
	bgBottomEnd := float64(ScreenHeight)
	bgBottom := bgBottomEnd
	bgTop := (bgBottom - float64(ScreenHeight))

	if t-activationTime < AnimationDuration { // Enter animation is in progress.
		diff := t - activationTime
		percentageCompleted := float64(diff) / float64(AnimationDuration)

		// bg calcs
		bgBottom = bgBottomStart - ((bgBottomStart - bgBottomEnd) * ease.OutElastic(percentageCompleted))
		bgTop = (bgBottom - float64(ScreenHeight))

		// text calcs
		textTop = textTopStart - ((textTopStart - textTopEnd) * ease.OutElastic(percentageCompleted))

	} else if t > activationTime+visibleLifetimeMS { // Exit animation in progress.
		diff := t - (activationTime + visibleLifetimeMS)
		percentageCompleted := float64(diff) / float64(AnimationDuration)

		// bg calcs
		bgBottom = bgBottomEnd - ((bgBottomEnd - bgBottomStart) * ease.InElastic(percentageCompleted))
		bgTop = (bgBottom - float64(ScreenHeight))

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
	ctx.SetFillColor(nanovgo.RGBA(122, 255, 122, 222))
	ctx.Fill()

	ctx.SetFontSize(96.0)
	ctx.SetFontFace("sans-bold")
	ctx.SetTextAlign(nanovgo.AlignCenter | nanovgo.AlignMiddle)

	// ctx.SetFontBlur(0)
	ctx.SetFillColor(nanovgo.RGBA(255, 255, 255, 255))
	middle := (float32(mgr.GetPresentationData().GlobalViewportSize) / 2) - float32(mgr.GetPresentationData().ScreenGlobalOffset)
	ctx.Text(middle, float32(textTop), message)
}

func renderTicker(ctx *nanovgo.Context, mgr tickerManager.TickerManager, ticker *tickerManager.Ticker, globalOffset int) {
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

	ctx.TextBox(float32(tickerOffset), 40, 900, ticker.Ticker+" $"+fmt.Sprintf("%.2f", ticker.Price))
	ctx.SetFontSize(56)
	ctx.SetFontFace("sans-light")
	ctx.TextBox(float32(tickerOffset), 180, 900, ticker.CompanyName)

}
