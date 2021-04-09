package main

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/goxjs/gl"
	"github.com/goxjs/glfw"
	tickerManager "github.com/polygon-io/go-app-ticker-wall/ticker_manager"
	"github.com/polygon-io/nanovgo"
	"github.com/polygon-io/nanovgo/perfgraph"
)

type Pos struct {
	sync.RWMutex
	left float32
}

const (
	ScreenWidth      = 1920
	ScreenHeight     = 300
	TargetFPS        = 90
	DurationPerFrame = time.Second / TargetFPS
)

func main() {
	err := glfw.Init(gl.ContextWatcher)
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	// Fullscreem:
	// window, err := glfw.CreateWindow(ScreenWidth, ScreenHeight, "Polygon Ticker Wall", glfw.GetPrimaryMonitor(), nil)
	// Windowed:
	window, err := glfw.CreateWindow(ScreenWidth, ScreenHeight, "Polygon Ticker Wall", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if key == glfw.KeyEscape {
			w.SetShouldClose(true)
		}
	})

	// ctx, err := nanovgo.NewContext(0)
	ctx, err := nanovgo.NewContext(0)
	defer ctx.Delete()
	if err != nil {
		panic(err)
	}

	glfw.SwapInterval(0)

	ctx.CreateFont("sans", "fonts/Roboto-Regular.ttf")

	pos := &Pos{
		left: float32(time.Now().UnixNano() / int64(time.Millisecond*15) % 1920),
	}
	go func() {
		for {
			pos.Lock()
			pos.left += 1
			pos.Unlock()
			time.Sleep(16 * time.Millisecond)
		}
	}()

	numbPtr := flag.Int("screenindex", 0, "Screen Index")
	flag.Parse()

	// Ticker Manager
	mgr := tickerManager.NewDefaultManager(&tickerManager.PresentationData{
		ScreenWidth:        ScreenWidth,
		ScreenHeight:       ScreenHeight,
		ScreenGlobalOffset: (ScreenWidth * *numbPtr),
		TickerBoxWidth:     1000,
		ScreenIndex:        *numbPtr,
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

	gl.ClearColor(0, 0, 0, 0)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.BLEND)
	gl.Disable(gl.CULL_FACE)
	gl.Disable(gl.DEPTH_TEST)

	for !window.ShouldClose() {
		start := time.Now()
		fps.UpdateGraph()
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)

		ctx.BeginFrame(winWidth, winHeight, pixelRatio)
		//ctx.Save()

		t := int(time.Now().UnixNano() / int64(time.Millisecond*15))
		// println(t)
		// Actual application drawing.
		pos.RLock()
		renderTickers(ctx, mgr, t)
		pos.RUnlock()

		//ctx.Restore()
		fps.RenderGraph(ctx, 5, 5)
		ctx.EndFrame()
		window.SwapBuffers()
		glfw.PollEvents()
		// time.Sleep(time.Millisecond * 16)
		time.Sleep(DurationPerFrame - time.Since(start))
	}

}

func renderTickers(ctx *nanovgo.Context, mgr tickerManager.TickerManager, globalOffset int) {
	tickers := mgr.DetermineTickersForRender(globalOffset)
	for _, ticker := range tickers {
		renderTicker(ctx, mgr, ticker, globalOffset)
	}
}

func renderTicker(ctx *nanovgo.Context, mgr tickerManager.TickerManager, ticker *tickerManager.Ticker, globalOffset int) {
	ctx.SetFontFace("sans")
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

	ctx.TextBox(float32(tickerOffset), 50, 900, ticker.Ticker+" $"+fmt.Sprintf("%.2f", ticker.Price))
	ctx.SetFontSize(56)
	ctx.TextBox(float32(tickerOffset), 190, 900, ticker.CompanyName)

}
