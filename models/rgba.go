package models

import "github.com/polygon-io/nanovgo"

func (g *RGBA) ToNanov() nanovgo.Color {
	return nanovgo.RGBA(
		uint8(g.Red),
		uint8(g.Green),
		uint8(g.Blue),
		uint8(g.Alpha),
	)
}
