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

	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
)

// clusterCmd represents the cluster command
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Operate on kr8 clusters",
	Long:  `Manage, list and generate kr8 cluster configurations at the cluster scope`,
	//Run: func(cmd *cobra.Command, args []string) { },
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Clusters",
	Long:  "List Clusters in kr8 config hierarchy",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := GetConfigFromContext(cmd.Context())

		clusters, err := getClusters(cfg)

		if err != nil {
			log.Fatal().Err(err).Msg("Error getting cluster")
		}

		var entry []string
		table := tablewriter.NewWriter(os.Stdout)
		table.Header([]string{"Name", "Path"})

		for _, c := range clusters.Cluster {
			entry = append(entry, c.Name)
			entry = append(entry, c.Path)
			table.Append(entry)
			entry = entry[:0]
		}
		table.Render()

	},
}

var paramsCmd = &cobra.Command{
	Use:   "params",
	Short: "Show Cluster Params",
	Long:  "Show cluster params in kr8 config hierarchy",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := GetConfigFromContext(cmd.Context())

		clusterName, _ := cmd.Flags().GetString("cluster")
		clusterParams, _ := cmd.Flags().GetString("clusterparams")

		if clusterName == "" && clusterParams == "" {
			log.Fatal().Msg("Please specify a --cluster name and/or --clusterparams")
		}

		componentName, _ := cmd.Flags().GetString("component")
		var clist []string
		if componentName != "" {
			clist = append(clist, componentName)
		}
		j := renderClusterParams(cfg, cmd, clusterName, clist, clusterParams, false)

		paramPath, _ := cmd.Flags().GetString("param")
		if paramPath != "" {
			value := gjson.Get(j, paramPath)
			notunset, _ := cmd.Flags().GetBool("notunset")
			if notunset && value.String() == "" {
				log.Fatal().Msg("Error getting param: " + paramPath)
			} else {
				fmt.Println(value) // no formatting because this isn't always json, this is just the value of a field
			}
		} else {
			formatted := Pretty(j, cfg.ColorOutput)
			fmt.Println(formatted)
		}

	},
}

var componentsCmd = &cobra.Command{
	Use:   "components",
	Short: "Show Cluster Components",
	Long:  "Show the components to be installed in the cluster in the kr8 hierarchy",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := GetConfigFromContext(cmd.Context())

		clusterName, _ := cmd.Flags().GetString("cluster")
		clusterParams, _ := cmd.Flags().GetString("clusterparams")

		if clusterName == "" && clusterParams == "" {
			log.Fatal().Msg("Please specify a --cluster name and/or --clusterparams")
		}

		var params []string
		if clusterName != "" {
			clusterPath := getCluster(cfg, clusterName)
			params = getClusterParams(cfg, clusterPath)
		}
		if clusterParams != "" {
			params = append(params, clusterParams)
		}

		j := renderJsonnet(cfg, cmd, params, "._components", true, "", "clustercomponents")
		paramPath, _ := cmd.Flags().GetString("param")
		if paramPath != "" {
			value := gjson.Get(j, paramPath)
			if value.String() == "" {
				log.Fatal().Msg("Error getting param: " + paramPath)
			} else {
				formatted := Pretty(j, cfg.ColorOutput)
				fmt.Println(formatted)
			}
		} else {
			formatted := Pretty(j, cfg.ColorOutput)
			fmt.Println(formatted)
		}

	},
}

func init() {
	RootCmd.AddCommand(clusterCmd)
	clusterCmd.AddCommand(listCmd)
	clusterCmd.AddCommand(paramsCmd)
	clusterCmd.AddCommand(componentsCmd)
	clusterCmd.PersistentFlags().StringP("cluster", "c", "", "cluster to operate on")
	clusterCmd.PersistentFlags().String("clusterparams", "", "provide cluster params as single file - can be combined with --cluster to override cluster")
	paramsCmd.PersistentFlags().String("component", "", "component to render params for")
	paramsCmd.Flags().StringP("param", "P", "", "return value of json param from supplied path")
	paramsCmd.Flags().BoolP("notunset", "", false, "Fail if specified param is not set. Otherwise returns blank value if param is not set")
	componentsCmd.Flags().StringP("param", "P", "", "return value of json param from supplied path")
}
