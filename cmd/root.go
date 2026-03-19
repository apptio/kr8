// Copyright © 2018 Lee Briggs <lee@leebriggs.co.uk>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// exported Version variable - kept as global since it's set once at startup
var Version string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "kr8",
	Short: "Kubernetes config parameter framework",
	Long: `A tool to generate Kubernetes configuration from a hierarchy
	of jsonnet files`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Build config from flags and viper
		cfg := buildConfigFromFlags(cmd)

		// Store config in command context for child commands
		ctx := SetConfigInContext(cmd.Context(), cfg)
		cmd.SetContext(ctx)

		return nil
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(version string) {
	Version = version
	// Initialize context for the root command
	ctx := context.Background()
	RootCmd.SetContext(ctx)

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Define flags
	RootCmd.PersistentFlags().StringP("base", "d", ".", "kr8 config base directory")
	RootCmd.PersistentFlags().StringP("clusterdir", "D", "", "kr8 cluster directory")
	RootCmd.PersistentFlags().StringP("componentdir", "X", "", "kr8 component directory")
	RootCmd.PersistentFlags().StringP("loglevel", "L", "info", "set log level")
	RootCmd.PersistentFlags().Bool("debug", false, "log more information about what kr8 is doing. Overrides --loglevel")
	RootCmd.PersistentFlags().Bool("color", true, "enable colorized output (default). Set to false to disable")
	RootCmd.PersistentFlags().StringArrayP("jpath", "J", nil, "Directories to add to jsonnet include path. Repeat arg for multiple directories")
	RootCmd.PersistentFlags().StringSlice("ext-str-file", nil, "Set jsonnet extvar from file contents")
}

// initConfig sets up logging based on flags
// This is called before command execution via cobra.OnInitialize
func initConfig() {
	// Setup logging based on flags
	debug, _ := RootCmd.PersistentFlags().GetBool("debug")
	logLevel, _ := RootCmd.PersistentFlags().GetString("loglevel")

	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		switch logLevel {
		case "debug":
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		case "info":
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		case "warn":
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		case "error":
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		case "fatal":
			zerolog.SetGlobalLevel(zerolog.FatalLevel)
		case "panic":
			zerolog.SetGlobalLevel(zerolog.PanicLevel)
		default:
			log.Fatal().Msg("invalid log level: " + logLevel)
		}
	}

	// Setup console output with color
	colorOutput, _ := RootCmd.PersistentFlags().GetBool("color")
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:     os.Stderr,
		NoColor: !colorOutput,
	})
}

// buildConfigFromFlags creates a Config struct from command flags
func buildConfigFromFlags(cmd *cobra.Command) *Config {
	cfg := NewConfig()

	// Get values from flags
	cfg.BaseDir, _ = cmd.Flags().GetString("base")
	cfg.ClusterDir, _ = cmd.Flags().GetString("clusterdir")
	cfg.ComponentDir, _ = cmd.Flags().GetString("componentdir")
	cfg.ColorOutput, _ = cmd.Flags().GetBool("color")
	cfg.JsonnetPaths, _ = cmd.Flags().GetStringArray("jpath")
	cfg.ExtVarFiles, _ = cmd.Flags().GetStringSlice("ext-str-file")

	// Set defaults for directories if not specified
	if cfg.ClusterDir == "" {
		cfg.ClusterDir = cfg.BaseDir + "/clusters"
	}
	if cfg.ComponentDir == "" {
		cfg.ComponentDir = cfg.BaseDir + "/components"
	}

	log.Debug().Msg("Using base directory: " + cfg.BaseDir)
	log.Debug().Msg("Using cluster directory: " + cfg.ClusterDir)
	log.Debug().Msg("Using component directory: " + cfg.ComponentDir)

	return cfg
}
