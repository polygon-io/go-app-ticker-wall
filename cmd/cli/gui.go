package main

import (
	"os"

	"github.com/polygon-io/go-app-ticker-wall/gui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newGUICmd() *cobra.Command {
	cfg := &gui.Config{}

	cmd := &cobra.Command{
		Use:   "gui",
		Short: `Start a new instance of the GUI.`,
		Long:  `Start a new instance of the GUI.`,
		Run: func(cmd *cobra.Command, args []string) {
			leader, _ := cmd.Flags().GetString("leader")
			cfg.ClientConfig.Leader = leader

			// Actually start the GUI process.
			if err := gui.Run(cfg); err != nil {
				logrus.WithError(err).Error("GUI encountered an error.")
				os.Exit(1)
			}
		},
	}

	cmd.Flags().IntVarP(&cfg.ClientConfig.ScreenHeight, "screen-height", "y", 300, "Height of this GUI window, in pixels.")
	cmd.Flags().IntVarP(&cfg.ClientConfig.ScreenWidth, "screen-width", "x", 1600, "Width of this GUI window, in pixels.")
	cmd.Flags().IntVarP(&cfg.ClientConfig.ScreenIndex, "screen-index", "i", 10, "Index of this GUI window in the window array. Eg: First screen: 10, Second screen: 20, and so on. This is an arbitrary number, used for sorting order.")

	return cmd
}
