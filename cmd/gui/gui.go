package main

import (
	"context"
	"fmt"
	"time"

	"github.com/goxjs/gl"
	"github.com/goxjs/glfw"
	"github.com/polygon-io/go-app-ticker-wall/client"
	"github.com/polygon-io/go-app-ticker-wall/fonts"
	"github.com/polygon-io/nanovgo"
	"github.com/polygon-io/nanovgo/perfgraph"
	"github.com/sirupsen/logrus"
)

type GUI struct {
	client client.Client

	// nanov
	window   *glfw.Window
	nanoCtx  *nanovgo.Context
	fpsGraph *perfgraph.PerfGraph

	//
	windowHeight int
	windowWidth  int
	pixelRatio   float32
}

func NewGUI(client client.Client) *GUI {
	obj := &GUI{
		client: client,
	}
	return obj
}

func (g *GUI) Setup() error {
	// Init glfw.
	if err := glfw.Init(gl.ContextWatcher); err != nil {
		return err
	}

	// Get our current screen state.
	screen := g.client.GetScreen()

	// Create a new window.
	window, err := glfw.CreateWindow(
		int(screen.Width),
		int(screen.Height),
		fmt.Sprintf("Polygon Ticker Wall %d", screen.Index),
		nil, nil,
	)
	if err != nil {
		return err
	}
	g.window = window
	g.window.MakeContextCurrent()
	g.window.SetCloseCallback(func(w *glfw.Window) {
		logrus.Info("Window Closed")
	})

	// Create context
	nanoCtx, err := nanovgo.NewContext(0)
	if err != nil {
		return err
	}
	g.nanoCtx = nanoCtx

	// This limits the refresh rate to that of the display.
	glfw.SwapInterval(1)

	// Load in fonts to our context.
	fonts.CreateFonts(g.nanoCtx)

	// Set viewport and pixel ratio.
	fbWidth, fbHeight := g.window.GetFramebufferSize()
	g.windowWidth, g.windowHeight = g.window.GetSize()
	g.pixelRatio = float32(fbWidth) / float32(g.windowWidth)
	gl.Viewport(0, 0, fbWidth, fbHeight)

	// Create FPS graph.
	g.fpsGraph = perfgraph.NewPerfGraph("Frame Time", "sans")

	// Some additional settings. Don't really know what these mean, using what nanovgo repo code had.
	gl.Enable(gl.BLEND)
	gl.Disable(gl.CULL_FACE)
	gl.Disable(gl.DEPTH_TEST)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	g.nanoCtx.SetFontFace("sans")
	g.nanoCtx.SetTextAlign(nanovgo.AlignLeft | nanovgo.AlignTop)
	g.nanoCtx.SetTextLineHeight(1.2)

	return nil
}

func (g *GUI) Close() error {
	g.nanoCtx.Delete()
	glfw.Terminate()
	return nil
}

func (g *GUI) Run(ctx context.Context) error {
	return nil
}

func (g *GUI) RenderLoop(ctx context.Context) error {
	// Load in the company images, and assign to each ticker.
	// for _, ticker := range mgr.AllTickers() {
	// 	img := g.nanoCtx.CreateImage("./logos/"+ticker.Ticker.Ticker+".png", 0)
	// 	g.nanoCtx.CreateImageFromMemory()
	// 	ticker.Ticker.Img = int32(img)
	// }

	// This is the main rendering loop. Every frame rendered must run everything in this loop.
	for !g.window.ShouldClose() {
		g.fpsGraph.UpdateGraph()

		if g.client.GetCluster() == nil {
			// This should be displayed on the app using a new system message method.
			logrus.Debug("Cluster not ready yet. Waiting on gRPC..")
			time.Sleep(1 * time.Second)
			continue
		}

		// Clear
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)
		g.nanoCtx.BeginFrame(g.windowWidth, g.windowHeight, g.pixelRatio)

		globalOffsetTimestamp := g.generateGlobalOffset()

		// Set BG color
		g.paintBG()

		// Actual application drawing.
		if err := g.renderTickers(globalOffsetTimestamp); err != nil {
			return err
		}

		// // If we have an announcement, display it.
		// if tickerWallClient.announcement != nil {
		// 	renderSpecialMessage(g.nanoCtx, mgr, t, tickerWallClient.announcement)
		// }

		g.renderFPSGraph()
		g.nanoCtx.EndFrame()
		gl.Enable(gl.DEPTH_TEST)
		g.window.SwapBuffers()
		glfw.PollEvents()
	}

	return ctx.Err()
}

// renderFPSGraph always renders the graph, but this decides if it should be displayed
// visibly. Removing the graph caused a massive memory leak.
// TODO: Find/Fix the memory leak so we don't always have to display the graph.
func (g *GUI) renderFPSGraph() {
	settings := g.client.GetSettings()

	if settings.ShowFPS {
		g.fpsGraph.RenderGraph(g.nanoCtx, 0, 0)
	} else {
		g.fpsGraph.RenderGraph(g.nanoCtx, -50, -50)
	}
}

var offset int64

// generateGlobalOffset generates the pixel offset taking into account the scroll speed.
func (g *GUI) generateGlobalOffset() int64 {
	// settings := g.client.GetSettings()

	// return time.Now().UnixNano() / int64(settings.ScrollSpeed*int32(time.Millisecond))
	// return time.Now().UnixNano() / int64(time.Millisecond)
	offset++
	return offset
}

// paintBG sets the background of the window to a solid color.
func (g *GUI) paintBG() {
	settings := g.client.GetSettings()

	// Set BG color
	g.nanoCtx.BeginPath()
	g.nanoCtx.RoundedRect(0, 0, float32(g.windowWidth), float32(g.windowHeight), 0)
	g.nanoCtx.SetFillColor(nanovgo.RGBA(
		uint8(settings.BGColor.Red),
		uint8(settings.BGColor.Green),
		uint8(settings.BGColor.Blue),
		uint8(settings.BGColor.Alpha),
	))
	g.nanoCtx.Fill()
}
