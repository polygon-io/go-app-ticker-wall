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
	// The name of our config file, without the file extension because viper supports many different config file languages.
	defaultConfigFilename = "stingoftheviper"

	// The environment variable prefix of all environment variables bound to our command line flags.
	// For example, --number is bound to STING_NUMBER.
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
	// apikey := ""

	// Define our command
	rootCmd := &cobra.Command{
		Use:   "tickerwall",
		Short: "Polygon.io Ticker Wall",
		Long: `A horizontally scalable ticker wall to display real-time stock data. 
Find out more at: https://github.com/polygon-io/go-app-ticker-wall`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeConfig(cmd)
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Working with OutOrStdout/OutOrStderr allows us to unit test our command easier
			out := cmd.OutOrStdout()
			key, err := cmd.Flags().GetString("api-key")
			if err != nil {
				panic(err)
			}

			// Print the final resolved value from binding cobra flags and viper config
			fmt.Fprintln(out, "Your apikey is:", key)
		},
	}

	// rootCmd.PersistentFlags().StringVarP(&apikey, "api-key", "a", "", "Your Polygon.io API Key. This key will be used to access Polygon.io for data.")
	rootCmd.PersistentFlags().StringP("api-key", "a", "", "Your Polygon.io API Key. This key will be used to access Polygon.io for data.")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Debug enables more verbose logging.")

	// Add additional commands
	rootCmd.AddCommand(newGUICmd())
	rootCmd.AddCommand(newServerCmd())

	return rootCmd
}

func initializeConfig(cmd *cobra.Command) error {
	v := viper.New()

	// Set the base name of the config file, without the file extension.
	v.SetConfigName(defaultConfigFilename)

	// Set as many paths as you like where viper should look for the
	// config file. We are only looking in the current working directory.
	v.AddConfigPath(".")

	// Attempt to read the config file, gracefully ignoring errors
	// caused by a config file not being found. Return an error
	// if we cannot parse the config file.
	if err := v.ReadInConfig(); err != nil {
		// It's okay if there isn't a config file
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	// When we bind flags to environment variables expect that the
	// environment variables are prefixed, e.g. a flag like --number
	// binds to an environment variable STING_NUMBER. This helps
	// avoid conflicts.
	v.SetEnvPrefix(envPrefix)

	// Bind to environment variables
	// Works great for simple config names, but needs help for names
	// like --favorite-color which we fix in the bindFlags function
	v.AutomaticEnv()

	// Bind the current command's flags to viper
	bindFlags(cmd, v)

	return nil
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Environment variables can't have dashes in them, so bind them to their equivalent
		// keys with underscores, e.g. --favorite-color to STING_FAVORITE_COLOR
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
