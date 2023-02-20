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
    "fmt"
    "os"
    "strconv"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"

    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
)

var (
    cfgFile       string
    baseDir       string
    clusterDir    string
    componentDir  string
    clusterParams string
    cluster       string

    debug       bool
    colorOutput bool
    noexit      bool
    long        bool
    warn        bool
    err         error = nil

    exit int = 0
)

// exported Version variable
var Version string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
    Use:   "kr8",
    Short: "Kubernetes config parameter framework",
    Long: `A tool to generate Kubernetes configuration from a hierarchy
	of jsonnet files`,
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(version string) {
    Version = version
    if err := RootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(-1)
    }
    if noexit {
        // Pass any exit codes declared during processing when `noexit` is enabled
        os.Exit(exit)
    }
}

func init() {
    cobra.OnInitialize(initConfig)

    RootCmd.PersistentFlags().StringVarP(&baseDir, "base", "d", ".", "kr8 config base directory")
    RootCmd.PersistentFlags().StringVarP(&clusterDir, "clusterdir", "D", "", "kr8 cluster directory")
    RootCmd.PersistentFlags().StringVarP(&componentDir, "componentdir", "X", "", "kr8 component directory")
    RootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "log more information about what kr8 is doing")
    RootCmd.PersistentFlags().BoolVar(&warn, "warn", false, "log less information about what kr8 is doing")
    RootCmd.PersistentFlags().BoolVar(&colorOutput, "color", true, "enable colorized output (default). Set to false to disable")
    RootCmd.PersistentFlags().BoolVar(&noexit, "noexit", false, "No exit when encountering a fatal error")
    RootCmd.PersistentFlags().BoolVar(&long, "long", false, "long-form error messages")
    RootCmd.PersistentFlags().StringArrayP("jpath", "J", nil, "Directories to add to jsonnet include path. Repeat arg for multiple directories")
    RootCmd.PersistentFlags().StringSlice("ext-str-file", nil, "Set jsonnet extvar from file contents")
    viper.BindPFlag("base", RootCmd.PersistentFlags().Lookup("base"))
    viper.BindPFlag("clusterdir", RootCmd.PersistentFlags().Lookup("clusterdir"))
    viper.BindPFlag("componentdir", RootCmd.PersistentFlags().Lookup("componentdir"))
    viper.BindPFlag("color", RootCmd.PersistentFlags().Lookup("color"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
    if cfgFile != "" { // enable ability to specify config file via flag
        viper.SetConfigFile(cfgFile)
    }

    viper.SetConfigName(".kr8") // name of config file (without extension)
    viper.AddConfigPath(".")
    viper.AddConfigPath("$HOME") // adding home directory as first search path
    viper.SetEnvPrefix("KR8")
    viper.AutomaticEnv() // read in environment variables that match

    // If a config file is found, read it in.
    if err := viper.ReadInConfig(); err == nil {
        infolog(err).Msg("Using config file:" + viper.ConfigFileUsed())
    }
    colorOutput = viper.GetBool("color")
    log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: !colorOutput})

    if debug {
        zerolog.SetGlobalLevel(zerolog.DebugLevel)
    } else {
        if warn {
            infolog(nil).Msg("Info level and lower logs will be suppressed. Using '--debug' overrides '--warn'")
            zerolog.SetGlobalLevel(zerolog.WarnLevel)
        } else {
            zerolog.SetGlobalLevel(zerolog.InfoLevel)
        }
    }
    baseDir = viper.GetString("base")
    debuglog(err).Msg("Using base directory: " + baseDir)
    clusterDir = viper.GetString("clusterdir")
    if clusterDir == "" {
        clusterDir = baseDir + "/clusters"
    }
    debuglog(err).Msg("Using cluster directory: " + clusterDir)
    if componentDir == "" {
        componentDir = baseDir + "/components"
    }
    debuglog(err).Msg("Using component directory: " + componentDir)

    debuglog(err).Msg("No Exit when Fatal Event occurs: " + strconv.FormatBool(noexit))

    debuglog(nil).Msg("Long pretty errors and messages: " + strconv.FormatBool(long))

}
