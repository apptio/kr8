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
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
	"os"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
)

var (
	clusterName   string
	componentName string
	paramPath     string
	formatted     string
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

		clusters, err := getClusters(base)

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

var paramsCmd = &cobra.Command{
	Use:   "params",
	Short: "Show Cluster Params",
	Long:  "Show cluster params in kr8 config hierarchy",
	Run: func(cmd *cobra.Command, args []string) {

		if clusterName == "" {
			log.Fatal("Please specify a cluster name")
		}
		clusterPath := getCluster(base, clusterName)

		params := getClusterParams(base, clusterPath)

		// render without component first
		j := renderJsonnet(cmd, params, "", true, "")
		if componentName != "" {
			// lookup the configured path for this component
			componentPrefix := gjson.Get(j, "_components."+componentName+".path")
			if componentPrefix.String()  == "" {
				log.Fatal("Component is not defined for this cluster: ", componentName)
			}
			componentPath := base + "/" + componentPrefix.String() + "/params.jsonnet"
			if _, err := os.Stat(componentPath); os.IsNotExist(err) {
				log.Fatal("No component found at: ", componentPath)
			}

			// we read the params.jsonnet for the component and append the code into the snippet
			// with the field name set to the componentName
			filec, err := ioutil.ReadFile(componentPath)
			if err != nil {
				log.Panic("Error reading file:", err)
			}

			prepend :=  "{"+componentName+": "+string(filec)+ "}"
			j = renderJsonnet(cmd, params, "", true, prepend)
		}
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

var componentsCmd = &cobra.Command{
	Use:   "components",
	Short: "Show Cluster Components",
	Long:  "Show the components to be installed in the cluster in the kr8 hierarchy",
	Run: func(cmd *cobra.Command, args []string) {

		if clusterName == "" {
			log.Fatal("Please specify a cluster name")
		}
		clusterPath := getCluster(base, clusterName)

		params := getClusterParams(base, clusterPath)
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

func init() {
	RootCmd.AddCommand(clusterCmd)
	clusterCmd.AddCommand(listCmd)
	clusterCmd.AddCommand(paramsCmd)
	clusterCmd.AddCommand(componentsCmd)
	clusterCmd.PersistentFlags().StringVarP(&clusterName, "cluster", "c", "", "cluster to operate on")
	paramsCmd.PersistentFlags().StringVarP(&componentName, "component", "C", "", "component to render params for")
	paramsCmd.Flags().StringVarP(&paramPath, "param", "P", "", "return value of json param from supplied path")
	paramsCmd.Flags().BoolP("notunset", "", false, "Fail if specified param is not set. Otherwise returns blank value if param is not set")

}
