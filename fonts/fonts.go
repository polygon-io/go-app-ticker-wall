// Package fonts is This is a simple wrapper around the fonts in this directly. This reduces the complexity
// of the root application from having to know which directory it needs to be in, as well
// as copying around font files when building/running.
package fonts

import (
	// We are using embed to prevent loading files from disk, which simplifies everything.
	_ "embed"

	"github.com/polygon-io/nanovgo"
)

// nolint:gochecknoglobals // not sure how else to go about this.
//go:embed Roboto-Regular.ttf
var fontsRobotoRegular []byte

// nolint:gochecknoglobals // not sure how else to go about this.
//go:embed Roboto-Light.ttf
var fontsRobotoLight []byte

// nolint:gochecknoglobals // not sure how else to go about this.
//go:embed Roboto-Bold.ttf
var fontsRobotoBold []byte

// CreateFonts attaches the fonts to the nanovgo context.
func CreateFonts(ctx *nanovgo.Context) {
	ctx.CreateFontFromMemory("sans", fontsRobotoRegular, 1)
	ctx.CreateFontFromMemory("sans-light", fontsRobotoLight, 1)
	ctx.CreateFontFromMemory("sans-bold", fontsRobotoBold, 1)
}
