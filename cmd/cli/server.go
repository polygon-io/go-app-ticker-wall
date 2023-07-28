package main

import (
	"os"

	"github.com/polygon-io/go-app-ticker-wall/leader"
	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/polygon-io/go-app-ticker-wall/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newServerCmd() *cobra.Command {
	cfg := &server.ServiceConfig{
		LeaderConfig: leader.Config{
			Presentation: &models.PresentationSettings{},
		},
	}
	colorMap := &colorMap{}

	cmd := &cobra.Command{
		Use:   "server",
		Short: `Start a new instance of the Server.`,
		Long:  `Start a new instance of the Server.`,
		Run: func(cmd *cobra.Command, args []string) {
			parseColorMap(colorMap, cfg.LeaderConfig.Presentation)

			// Set the api key.
			apiKey, _ := cmd.Flags().GetString("api-key")
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

	cmd.Flags().StringVarP(&cfg.LeaderConfig.TickerList, "tickers", "t", "AAPL,AMD,NVDA,SBUX,META,HOOD", "A comma separated list of tickers to display on the ticker wall.")

	// Ports
	cmd.Flags().IntVarP(&cfg.GRPCPort, "grpc-port", "g", 6886, "Which port the GRPC Server should bind to.")
	cmd.Flags().IntVarP(&cfg.HTTPPort, "http-port", "p", 6887, "Which port the HTTP Server should bind to.")

	// Presentation Settings.
	cmd.Flags().AddFlagSet(presentationFlags(cfg.LeaderConfig.Presentation))

	// Color Settings.
	cmd.Flags().AddFlagSet(colorFlags(colorMap))

	// Dont auto sort flags.
	cmd.Flags().SortFlags = false

	return cmd
}
