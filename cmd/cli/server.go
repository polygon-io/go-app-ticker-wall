package main

import (
	"os"

	"github.com/polygon-io/go-app-ticker-wall/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newServerCmd() *cobra.Command {
	cfg := &server.ServiceConfig{}
	colorMap := &colorMap{}

	cmd := &cobra.Command{
		Use:   "server",
		Short: `Start a new instance of the Server.`,
		Long:  `Start a new instance of the Server.`,
		Run: func(cmd *cobra.Command, args []string) {
			apiKey, _ := cmd.Flags().GetString("api-key")
			debug, _ := cmd.Flags().GetBool("debug")

			parseColorMap(colorMap, cfg)

			// Set the global options.
			cfg.Debug = debug
			cfg.LeaderConfig.APIKey = apiKey

			if cfg.LeaderConfig.APIKey == "" {
				logrus.Error("You must set a Polygon.io API Key. Use the '-a' param to set the key. Eg: tickerwall server -a MY_API_KEY.")
				os.Exit(1)
			}

			// Actually start the Server process.
			if err := server.Run(cfg); err != nil {
				logrus.WithError(err).Error("Server encountered an error.")
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVarP(&cfg.LeaderConfig.TickerList, "ticker-list", "t", "AAPL,AMD,NVDA,SBUX,FB,HOOD", "A comma separated list of tickers to display on the ticker wall.")

	// Ports
	cmd.Flags().IntVarP(&cfg.GRPCPort, "grpc-port", "g", 6886, "Which port the GRPC Server should bind to.")
	cmd.Flags().IntVarP(&cfg.HTTPPort, "http-port", "p", 6887, "Which port the HTTP Server should bind to.")

	// Presentation Settings.
	cmd.Flags().IntVarP(&cfg.LeaderConfig.Presentation.ScrollSpeed, "scroll-speed", "s", 15, "How fast the tickers scroll across the screen. This is inverted so 0 is the fastest possible.")
	cmd.Flags().IntVarP(&cfg.LeaderConfig.Presentation.TickerBoxWidthPx, "ticker-box-width", "w", 1100, "The size of the ticker box, in pixels.")
	cmd.Flags().IntVarP(&cfg.LeaderConfig.Presentation.AnimationDuration, "animation-duration", "", 500, "Animation during of notifications, in milliseconds.")
	cmd.Flags().BoolVarP(&cfg.LeaderConfig.Presentation.PerTickUpdates, "per-tick-updates", "", true, "If the ticker wall should update on every trade which happens. Setting to false limits it to update 1/sec.")

	// Color Settings.
	colorFlags := pflag.NewFlagSet("color", pflag.ContinueOnError)
	colorFlags.StringArrayVarP(&colorMap.UpColor, "up-color", "", []string{"51", "255", "51", "255"}, "RGBA color mapping for the 'up' color. Array must be in order. red,green,blue,alpha.")
	colorFlags.StringArrayVarP(&colorMap.DownColor, "down-color", "", []string{"255", "51", "51", "255"}, "RGBA color mapping for the 'down' color. Array must be in order. red,green,blue,alpha.")
	colorFlags.StringArrayVarP(&colorMap.FontColor, "font-color", "", []string{"255", "255", "255", "255"}, "RGBA color mapping for the 'font' color. Array must be in order. red,green,blue,alpha.")
	colorFlags.StringArrayVarP(&colorMap.TickerBoxBGColor, "ticker-bg-color", "", []string{"20", "20", "20", "255"}, "RGBA color mapping for the 'font' color. Array must be in order. red,green,blue,alpha.")
	colorFlags.StringArrayVarP(&colorMap.BGColor, "bg-color", "", []string{"1", "1", "1", "255"}, "RGBA color mapping for the 'bg' color. Array must be in order. red,green,blue,alpha.")
	cmd.Flags().AddFlagSet(colorFlags)

	// Dont auto sort flags.
	cmd.Flags().SortFlags = false

	return cmd
}
