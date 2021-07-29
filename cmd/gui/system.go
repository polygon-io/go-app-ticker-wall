package main

import (
	"github.com/polygon-io/go-app-ticker-wall/client"
	"github.com/polygon-io/nanovgo"
)

func (g *GUI) SystemPanel() {
	status := g.client.GetStatus()
	screen := g.client.GetScreen()

	systemDialogPanelHeight := 200
	systemDialogPadding := float32(20)

	fromTop := (screen.Height / 2) - (int32(systemDialogPanelHeight) / 2)

	// Set BG color.
	g.nanoCtx.BeginPath()
	g.nanoCtx.RoundedRect(systemDialogPadding, float32(fromTop), float32(g.windowWidth)-(systemDialogPadding*2), float32(systemDialogPanelHeight), 5)
	g.nanoCtx.SetFillColor(nanovgo.RGBA(255, 0, 0, 222))
	g.nanoCtx.Fill()

	// Set font settings.
	g.nanoCtx.SetFontFace("sans-bold")
	g.nanoCtx.SetTextAlign(nanovgo.AlignCenter | nanovgo.AlignMiddle)
	g.nanoCtx.SetTextLineHeight(1.2)
	g.nanoCtx.SetFontSize(32.0)
	g.nanoCtx.SetFillColor(nanovgo.RGBA(255, 255, 255, 255))

	message := "System Panel"
	if status.GRPCStatus == client.GRPCStatusReconnecting {
		message = "Reconnecting to Leader.."
	} else if status.GRPCStatus == client.GRPCStatusDisconnected {
		message = "Disconnected from Leader.."
	}

	g.nanoCtx.Text(float32(screen.Width)/2, float32(screen.Height)/2, message)
}
