package main

import (
	"context"

	"github.com/polygon-io/go-app-ticker-wall/gui"
	"github.com/polygon-io/go-app-ticker-wall/models"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	cfg := &gui.Config{}
	var leaderClient *ServerClient

	// Where the new settings will be put
	newSettings := &models.PresentationSettings{}
	colorMap := &colorMap{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: `Update settings of a currently running server.`,
		Long:  `Update settings of a currently running server.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Create a leader client...
			leader, _ := cmd.Flags().GetString("leader")
			leaderClient, err = NewServerClient(leader)
			if err != nil {
				return err
			}

			logrus.Debug("Connected to Leader.")

			parseColorMap(colorMap, newSettings)

			if _, err = leaderClient.client.UpdatePresentationSettings(context.Background(), newSettings); err != nil {
				return err
			}

			logrus.Info("Settings Updated.")

			return nil
		},
	}

	cmd.Flags().StringVarP(&cfg.ClientConfig.Leader, "leader", "l", "localhost:6886", "Location of the leader. ( hostname:grpcPort ). If running locally, use default.")

	// Presentation Settings.
	cmd.Flags().AddFlagSet(presentationFlags(newSettings))

	// Color Settings.
	cmd.Flags().AddFlagSet(colorFlags(colorMap))

	// Dont auto sort flags.
	cmd.Flags().SortFlags = false

	return cmd
}
