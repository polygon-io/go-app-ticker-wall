package gui

import (
	"github.com/goxjs/glfw"
	"github.com/sirupsen/logrus"
)

func (g *GUI) windowClosedEvent(w *glfw.Window) {
	logrus.Debug("Window Closed")
}

func (g *GUI) windowResizeEvent(w *glfw.Window, width int, height int) {
	g.windowHeight = height
	g.windowWidth = width
	// TODO: Add a 100ms (or something) debounce here so we don't update the cluster too often.
	g.client.UpdateScreen(width, height)
}
