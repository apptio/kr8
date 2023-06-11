package cmd

import (
	goyaml "github.com/ghodss/yaml"
	"github.com/spf13/cobra"

	"bufio"
	"fmt"
	"io"
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Render files",
	Long:  `Render files in jsonnet or UAML`,
}

var renderjsonnetCmd = &cobra.Command{
	Use:   "jsonnet file [file ...]",
	Short: "Render a jsonnet file",
	Long:  `Render a jsonnet file to JSON or YAML`,

	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonnetrenderCmd.Run(cmd, args)
	},
}

var helmcleanCmd = &cobra.Command{
	Use:   "helmclean",
	Short: "Clean YAML stream from Helm Template output - Reads from Stdin",
	Long:  `Removes Null YAML objects from a YAML stream`,
	Run: func(cmd *cobra.Command, args []string) {
		decoder := yaml.NewYAMLReader(bufio.NewReader(os.Stdin))
		jsa := [][]byte{}
		for {
			bytes, err := decoder.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				fatalog(err).Msg("Error decoding decoding yaml stream")
			}
			if len(bytes) == 0 {
				continue
			}
			jsondata, err := yaml.ToJSON(bytes)
			if err != nil {
				fatalog(err).Msg("Error encoding yaml to JSON")
			}
			if string(jsondata) == "null" {
				// skip empty json
				continue
			}
			_, _, err = unstructured.UnstructuredJSONScheme.Decode(jsondata, nil, nil)
			if err != nil {
				fatalog(err).Msg("Error handling unstructured JSON")
			}
			jsa = append(jsa, jsondata)
		}
		for _, j := range jsa {
			out, err := goyaml.JSONToYAML(j)
			if err != nil {
				fatalog(err).Msg("Error encoding JSON to YAML")
			}
			fmt.Println("---")
			fmt.Println(string(out))
		}
	},
}

func init() {
	RootCmd.AddCommand(renderCmd)
	renderCmd.AddCommand(renderjsonnetCmd)
	renderjsonnetCmd.PersistentFlags().BoolVarP(&pruneFlag, "prune", "", true, "Prune null and empty objects from rendered json")
	renderjsonnetCmd.PersistentFlags().StringVarP(&clusterParams, "clusterparams", "", "", "provide cluster params as single file - can be combined with --cluster to override cluster")
	renderjsonnetCmd.PersistentFlags().StringVarP(&componentName, "component", "C", "", "component to render params for")
	renderjsonnetCmd.PersistentFlags().StringVarP(&outputFormat, "format", "F", "json", "Output forma: json, yaml, stream")
	renderjsonnetCmd.PersistentFlags().StringVarP(&cluster, "cluster", "c", "", "cluster to render params for")
	renderCmd.AddCommand(helmcleanCmd)
}
