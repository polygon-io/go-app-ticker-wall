package main

import (
	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/spf13/pflag"
)

// colorFlags creates a flagset for the color options.
func colorFlags(colorMap *colorMap) *pflag.FlagSet {
	colorFlags := pflag.NewFlagSet("color", pflag.ContinueOnError)
	colorFlags.StringVarP(&colorMap.UpColor, "up-color", "", "51,255,51,255", "RGBA color mapping for the 'up' color. Array must be in order. red,green,blue,alpha.")
	colorFlags.StringVarP(&colorMap.DownColor, "down-color", "", "255,51,51,255", "RGBA color mapping for the 'down' color. Array must be in order. red,green,blue,alpha.")
	colorFlags.StringVarP(&colorMap.FontColor, "font-color", "", "255,255,255,255", "RGBA color mapping for the 'font' color. Array must be in order. red,green,blue,alpha.")
	colorFlags.StringVarP(&colorMap.TickerBoxBGColor, "ticker-bg-color", "", "20,20,20,255", "RGBA color mapping for the 'font' color. Array must be in order. red,green,blue,alpha.")
	colorFlags.StringVarP(&colorMap.BGColor, "bg-color", "", "1,1,1,255", "RGBA color mapping for the 'bg' color. Array must be in order. red,green,blue,alpha.")
	return colorFlags
}

// presentationFlags creates a flagset for the presentation options.
func presentationFlags(presentationSettings *models.PresentationSettings) *pflag.FlagSet {
	presentationFlags := pflag.NewFlagSet("presentation", pflag.ContinueOnError)
	presentationFlags.Int32VarP(&presentationSettings.ScrollSpeed, "scroll-speed", "s", 15, "How fast the tickers scroll across the screen. This is inverted so 1 is the fastest possible.")
	presentationFlags.Int32VarP(&presentationSettings.TickerBoxWidth, "ticker-box-width", "w", 1100, "The size of the ticker box, in pixels.")
	presentationFlags.Int32VarP(&presentationSettings.AnimationDurationMS, "animation-duration", "", 500, "Animation during of notifications, in milliseconds.")
	presentationFlags.BoolVarP(&presentationSettings.PerTickUpdates, "per-tick-updates", "", true, "If the ticker wall should update on every trade which happens. Setting to false limits it to update 1/sec.")
	return presentationFlags
}
