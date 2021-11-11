package main

import (
	"context"
	"fmt"

	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newDescribeCmd() *cobra.Command {
	var leaderClient *ServerClient

	cmd := &cobra.Command{
		Use:   "describe",
		Short: `Describe a current running cluster.`,
		Long:  `Describe a current running cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Create a leader client...
			leader, _ := cmd.Flags().GetString("leader")
			leaderClient, err = NewServerClient(leader)
			if err != nil {
				return err
			}
			logrus.Debug("Connected to Leader.")

			cluster, err := leaderClient.client.GetScreenCluster(context.Background(), &models.Empty{})
			if err != nil {
				return err
			}

			tickers, err := leaderClient.client.GetTickers(context.Background(), &models.Empty{})
			if err != nil {
				return err
			}

			printClusterInfo(cluster, tickers)

			return nil
		},
	}

	// Dont auto sort flags.
	cmd.Flags().SortFlags = false

	return cmd
}

// printClusterInfo prints out the clusters details
// TODO: This would look a lot better as a table or something.
func printClusterInfo(cluster *models.ScreenCluster, tickers *models.Tickers) {
	fmt.Println("Global Viewport Size:", cluster.GlobalViewportSize(), "px")
	fmt.Println("Animation Duration:", cluster.Settings.AnimationDurationMS, "ms")
	fmt.Println("Scroll Speed:", cluster.Settings.ScrollSpeed)
	fmt.Println("Ticker Box Width:", cluster.Settings.TickerBoxWidth, "px")
	fmt.Println("Per Tick Updates:", cluster.Settings.PerTickUpdates)
	fmt.Println("Screen Count:", cluster.NumberOfScreens())
	fmt.Println("Screen Details:")
	for _, screen := range cluster.Screens {
		fmt.Println(" ------------ ")
		fmt.Println(" Screen ID:", screen.UUID)
		fmt.Println(" - Width", screen.Width, "px")
		fmt.Println(" - Height", screen.Height, "px")
		fmt.Println(" - Index", screen.Index)
	}
	fmt.Println(" ------------ ")
	fmt.Println("Ticker count:", len(tickers.Tickers))
	fmt.Println("Tickers:")

	for _, t := range tickers.Tickers {
		fmt.Println(" - ", t.Ticker, " [ ", t.CompanyName, " ]")
	}
}
