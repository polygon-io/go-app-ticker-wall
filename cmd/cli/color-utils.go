package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/polygon-io/go-app-ticker-wall/server"
	"github.com/sirupsen/logrus"
)

type colorMap struct {
	UpColor          []string `default:"red:51,green:255,blue:51,alpha:255"`
	DownColor        []string `default:"red:255,green:51,blue:51,alpha:255"`
	FontColor        []string `default:"red:255,green:255,blue:255,alpha:255"`
	TickerBoxBGColor []string `default:"red:20,green:20,blue:20,alpha:255"`
	BGColor          []string `default:"red:1,green:1,blue:1,alpha:255"`
}

func parseColorMap(cmap *colorMap, cfg *server.ServiceConfig) {
	cfg.LeaderConfig.Presentation.UpColor = mapColorArrayToMap(cmap.UpColor)
	cfg.LeaderConfig.Presentation.DownColor = mapColorArrayToMap(cmap.DownColor)
	cfg.LeaderConfig.Presentation.FontColor = mapColorArrayToMap(cmap.FontColor)
	cfg.LeaderConfig.Presentation.TickerBoxBGColor = mapColorArrayToMap(cmap.TickerBoxBGColor)
	cfg.LeaderConfig.Presentation.BGColor = mapColorArrayToMap(cmap.BGColor)
}

func mapColorArrayToMap(colors []string) map[string]int32 {
	mapping := make(map[string]int32)

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

	mapping["red"] = int32(red)
	mapping["green"] = int32(green)
	mapping["blue"] = int32(blue)
	mapping["alpha"] = int32(alpha)
	return mapping
}
