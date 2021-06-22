package main

import (
	"context"
	"fmt"
	"math"
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

	// logos keeps track of the logos loaded into render context.
	logos *LogoManager
}

func NewGUI(client client.Client) *GUI {
	obj := &GUI{
		client: client,
		logos:  NewLogosManager(),
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
		fmt.Sprintf("Polygon Ticker Wall ( INDEX: %d )", screen.Index),
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

	// Set the logo managers context.
	return g.logos.Setup(g.nanoCtx)
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
	// This is the main rendering loop. Every frame rendered must run everything in this loop.
	for !g.window.ShouldClose() {
		// Get frame ready.
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)
		g.nanoCtx.BeginFrame(g.windowWidth, g.windowHeight, g.pixelRatio)

		if err := g.renderFrame(); err != nil {
			logrus.WithError(err).Error("Could not render frame.")
		}

		g.endFrame()

		g.logos.RenderThread()
	}

	return ctx.Err()
}

func (g *GUI) renderFrame() error {
	// Get the client library status.
	status := g.client.GetStatus()

	// If we are having issues, display the system dialog panel
	if status.GRPCStatus != client.GRPCStatusConnected {
		// We use defer so that we render last, making sure we are displayed on top of all other content.
		defer func() {
			g.SystemPanel()
		}()
	}

	// Get cluster information.
	if g.client.GetCluster() == nil {
		// This should be displayed on the app using a new system message method.
		logrus.Debug("Cluster not ready yet. Waiting on gRPC..")
		time.Sleep(1 * time.Second)
		return nil
	}

	g.fpsGraph.UpdateGraph()

	globalOffsetTimestamp := g.generateGlobalOffset()

	// Set BG color
	g.paintBG()

	// Actual application drawing.
	if err := g.renderTickers(globalOffsetTimestamp); err != nil {
		return err
	}

	// TOOD: Render announcement if exists.

	g.renderFPSGraph()
	return nil
}

func (g *GUI) endFrame() {
	g.nanoCtx.EndFrame()
	gl.Enable(gl.DEPTH_TEST)
	g.window.SwapBuffers()
	glfw.PollEvents()
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

// generateGlobalOffset generates the pixel offset taking into account the scroll speed.
func (g *GUI) generateGlobalOffset() float32 {
	settings := g.client.GetSettings()
	tickers := g.client.GetTickers()

	newGlobalOffset := float64(time.Now().UnixNano()) / float64(int(settings.ScrollSpeed)*int(time.Millisecond))

	tickerBoxWidth := float32(settings.TickerBoxWidth)
	tapeWidth := float32(float32(len(tickers)) * tickerBoxWidth)
	baseDivisible := float64(math.Floor(float64(newGlobalOffset) / float64(tapeWidth)))
	newGlobalOffset = newGlobalOffset - (baseDivisible * float64(tapeWidth))

	return float32(newGlobalOffset)
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
