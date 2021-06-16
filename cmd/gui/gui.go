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
	client *client.Client

	// nanov
	window   *glfw.Window
	nanoCtx  *nanovgo.Context
	fpsGraph *perfgraph.PerfGraph

	//
	windowHeight int
	windowWidth  int
	pixelRatio   float32
}

func NewGUI(client *client.Client) *GUI {
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

	// Create a new window.
	window, err := glfw.CreateWindow(
		int(g.client.Screen.Width),
		int(g.client.Screen.Height),
		fmt.Sprintf("Polygon Ticker Wall %d", g.client.Screen.Index),
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

	// Some additional settings.
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

		if g.client.Cluster == nil {
			logrus.Debug("Cluster not ready yet. Check gRPC.")
			time.Sleep(1 * time.Second)
			continue
		}

		// Clear
		// gl.ClearColor(0, 0, 0, 0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)

		g.nanoCtx.BeginFrame(g.windowWidth, g.windowHeight, g.pixelRatio)

		// Set BG color
		g.nanoCtx.BeginPath()
		g.nanoCtx.RoundedRect(0, 0, float32(g.windowWidth), float32(g.windowHeight), 0)
		g.nanoCtx.SetFillColor(nanovgo.RGBA(
			uint8(g.client.Cluster.Settings.BGColor.Red),
			uint8(g.client.Cluster.Settings.BGColor.Green),
			uint8(g.client.Cluster.Settings.BGColor.Blue),
			uint8(g.client.Cluster.Settings.BGColor.Alpha),
		))
		g.nanoCtx.Fill()

		globalOffsetTimestamp := time.Now().UnixNano() / int64(g.client.Cluster.Settings.ScrollSpeed*int32(time.Millisecond))

		// Actual application drawing.
		if err := g.renderTickers(globalOffsetTimestamp); err != nil {
			return err
		}

		// // If we have an announcement, display it.
		// if tickerWallClient.announcement != nil {
		// 	renderSpecialMessage(g.nanoCtx,
		// 		mgr,
		// 		t,
		// 		tickerWallClient.announcement,
		// 	)
		// }

		g.fpsGraph.RenderGraph(g.nanoCtx, -50, -50)
		g.nanoCtx.EndFrame()
		gl.Enable(gl.DEPTH_TEST)
		g.window.SwapBuffers()
		glfw.PollEvents()
	}

	return ctx.Err()
}
