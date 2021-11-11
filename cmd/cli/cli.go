package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	defaultConfigFilename = "tickerwall"

	// The environment variable prefix of all environment variables bound to our command line flags.
	// For example, --debug is bound to TW_DEBUG.
	envPrefix = "TW"
)

func main() {
	cmd := NewRootCommand()
	if err := cmd.Execute(); err != nil {
		logrus.Error("ERR: ", err)
		os.Exit(1)
	}
}

// Build the cobra command that handles our command line tool.
func NewRootCommand() *cobra.Command {
	// Root command.
	rootCmd := &cobra.Command{
		Use:   "tickerwall",
		Short: "Polygon.io Ticker Wall",
		Long: `A horizontally scalable ticker wall to display real-time stock data. 
Find out more at: https://github.com/polygon-io/go-app-ticker-wall`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := initializeConfig(cmd); err != nil {
				return err
			}

			// Set log levels:
			debug, _ := cmd.Flags().GetBool("debug")
			logLevel := logrus.InfoLevel
			if debug {
				logLevel = logrus.DebugLevel
			}
			// Set Log Levels.
			logrus.SetLevel(logLevel)

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			println("Use the --help command to learn more about this apps abilities.")
		},
	}

	// Global flags.
	rootCmd.PersistentFlags().StringP("api-key", "a", "", "Your Polygon.io API Key. This key will be used to access Polygon.io for data.")
	rootCmd.PersistentFlags().StringP("leader", "l", "localhost:6886", "The leaders address of the cluster.")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Debug enables more verbose logging.")

	// Add additional commands.
	rootCmd.AddCommand(newGUICmd())
	rootCmd.AddCommand(newServerCmd())
	rootCmd.AddCommand(newUpdateCmd())
	rootCmd.AddCommand(newAnnounceCmd())
	rootCmd.AddCommand(newDescribeCmd())

	return rootCmd
}

func initializeConfig(cmd *cobra.Command) error {
	v := viper.New()

	v.SetConfigName(defaultConfigFilename)
	v.AddConfigPath(".")

	// Get home dir.
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	viper.AddConfigPath(home)

	if err := v.ReadInConfig(); err != nil {
		// It's okay if there isn't a config file
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	bindFlags(cmd, v)

	return nil
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Environment variables can't have dashes in them, so bind them to their equivalent
		// keys with underscores.
		if strings.Contains(f.Name, "-") {
			envVarSuffix := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
			v.BindEnv(f.Name, fmt.Sprintf("%s_%s", envPrefix, envVarSuffix))
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}
