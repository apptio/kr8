// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"

	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

var (
	cluster string
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Display one or many kr8 resources",
	Long:  `Displays information about kr8 resources such as clusters and components`,
}

var getclustersCmd = &cobra.Command{
	Use:   "clusters",
	Short: "Get all clusters",
	Long:  "Get all clusters defined in kr8 config hierarchy",
	Run: func(cmd *cobra.Command, args []string) {

		clusters, err := getClusters(clusterDir)

		if err != nil {
			log.Fatal("Error getting cluster: ", err)
		}

		var entry []string
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "Path"})

		for _, c := range clusters.Cluster {
			entry = append(entry, c.Name)
			entry = append(entry, c.Path)
			table.Append(entry)
			entry = entry[:0]
		}
		table.Render()

	},
}

var getcomponentsCmd = &cobra.Command{
	Use:   "components",
	Short: "Get all components",
	Long:  "Get all available components defined in the kr8 config hierarchy",
	Run: func(cmd *cobra.Command, args []string) {

		clusterName := cluster

		if clusterName == "" {
			log.Fatal("Please specify a --cluster name")
		}

		var params []string
		if clusterName != "" {
			clusterPath := getCluster(clusterDir, clusterName)
			params = getClusterParams(clusterDir, clusterPath)
		}
		if clusterParams != "" {
			params = append(params, clusterParams)
		}

		j := renderJsonnet(cmd, params, "._components", true, "")
		if paramPath != "" {
			value := gjson.Get(j, paramPath)
			if value.String() == "" {
				log.Fatal("Error getting param: ", paramPath)
			} else {
				formatted = Pretty(j, colorOutput)
				fmt.Println(formatted)
			}
		} else {
			formatted = Pretty(j, colorOutput)
			fmt.Println(formatted)
		}
	},
}

var getparamsCmd = &cobra.Command{
	Use:   "params",
	Short: "Get parameter for components and clusters",
	Long:  "Get parameters assigned to clusters and components in the kr8 config hierarchy",
	Run: func(cmd *cobra.Command, args []string) {

		clusterName := cluster

		if clusterName == "" {
			log.Fatal("Please specify a --cluster")
		}

		j := renderClusterParams(cmd, clusterName, componentName, clusterParams)

		if paramPath != "" {
			value := gjson.Get(j, paramPath)
			notunset, _ := cmd.Flags().GetBool("notunset")
			if notunset && value.String() == "" {
				log.Fatal("Error getting param: ", paramPath)
			} else {
				fmt.Println(value) // no formatting because this isn't always json, this is just the value of a field
			}
		} else {
			formatted = Pretty(j, colorOutput)
			fmt.Println(formatted)
		}

	},
}

func init() {
	RootCmd.AddCommand(getCmd)
	// clusters
	getCmd.AddCommand(getclustersCmd)
	// components
	getCmd.AddCommand(getcomponentsCmd)
	getcomponentsCmd.PersistentFlags().StringVarP(&cluster, "cluster", "c", "", "get components for cluster")
	// params
	getCmd.AddCommand(getparamsCmd)
	getparamsCmd.PersistentFlags().StringVarP(&cluster, "cluster", "c", "", "get components for cluster")

}
