package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
)

type colorMap struct {
	UpColor          string `default:"51,255,51,255"`
	DownColor        string `default:"255,51,51,255"`
	FontColor        string `default:"255,255,255,255"`
	TickerBoxBGColor string `default:"20,20,20,255"`
	BGColor          string `default:"1,1,1,255"`
}

func parseColorMap(cmap *colorMap, cfg *models.PresentationSettings) {
	cfg.UpColor = mapColorArrayToMap(cmap.UpColor)
	cfg.DownColor = mapColorArrayToMap(cmap.DownColor)
	cfg.FontColor = mapColorArrayToMap(cmap.FontColor)
	cfg.TickerBoxBGColor = mapColorArrayToMap(cmap.TickerBoxBGColor)
	cfg.BGColor = mapColorArrayToMap(cmap.BGColor)
}

func mapColorArrayToMap(colorString string) *models.RGBA {
	colors := strings.Split(colorString, ",")

	if len(colors) != 4 {
		logrus.Debug("Color mapping does not have enough attributes. Requires 4, has: ", len(colors), " Value: ", strings.Join(colors, ","))
		os.Exit(1)
	}

	red, err := strconv.Atoi(colors[0])
	if err != nil {
		logrus.Error("Got error decoding reds value: ", colors[0])
	}
	green, err := strconv.Atoi(colors[1])
	if err != nil {
		logrus.Error("Got error decoding green value: ", colors[0])
	}
	blue, err := strconv.Atoi(colors[2])
	if err != nil {
		logrus.Error("Got error decoding blue value: ", colors[0])
	}
	alpha, err := strconv.Atoi(colors[3])
	if err != nil {
		logrus.Error("Got error decoding alpha value: ", colors[0])
	}

	return &models.RGBA{
		Red:   int32(red),
		Green: int32(green),
		Blue:  int32(blue),
		Alpha: int32(alpha),
	}
}
