package main

import (
	"flag"
	"time"

	"github.com/goxjs/gl"
	"github.com/goxjs/glfw"
	tickerManager "github.com/polygon-io/go-app-ticker-wall/ticker_manager"
	"github.com/polygon-io/nanovgo"
	"github.com/polygon-io/nanovgo/perfgraph"
	"github.com/sirupsen/logrus"
)

var (
	// AnimationDuration is the length of animations.
	AnimationDuration = 750 // ms
)

func main() {
	screenIndexPtr := flag.Int("screenindex", 0, "Screen Index")
	screenWidthPtr := flag.Int("screenwidth", 1920, "Constant size of each screen width")
	screenHeightPtr := flag.Int("screenheight", 300, "Constant size of each screen height")
	totalScreensPtr := flag.Int("totalscreens", 1, "Total number of screens")
	flag.Parse()

	err := glfw.Init(gl.ContextWatcher)
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	window, err := glfw.CreateWindow(*screenWidthPtr, *screenHeightPtr, "Polygon Ticker Wall", nil, nil)
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

	// Ticker Manager
	mgr := tickerManager.NewDefaultManager(&tickerManager.PresentationData{
		ScreenWidth:        *screenWidthPtr,
		ScreenHeight:       *screenHeightPtr,
		ScreenGlobalOffset: int64(*screenWidthPtr * *screenIndexPtr),
		TickerBoxWidth:     1000,
		ScreenIndex:        *screenIndexPtr,
		NumberOfScreens:    *totalScreensPtr,
		GlobalViewportSize: int64(*totalScreensPtr * *screenWidthPtr),
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

		t := time.Now().UnixNano() / int64(time.Millisecond*10)
		// println(t)
		// Actual application drawing.
		renderTickers(ctx, mgr, t)

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
