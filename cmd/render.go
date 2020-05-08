package cmd

import (
	"encoding/json"

	goyaml "github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
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

		clusterName := cluster

		if clusterName == "" && clusterParams == "" {
			log.Fatal("Please specify a --cluster name and/or --clusterparams")
		}

		config := renderClusterParams(cmd, clusterName, componentName, clusterParams, false)

		// VM
		vm, _ := JsonnetVM(cmd)

		var input string
		// pass component, _cluster and _components as extvars
		vm.ExtCode("kr8_cluster", "std.prune("+config+"._cluster)")
		vm.ExtCode("kr8_components", "std.prune("+config+"._components)")
		vm.ExtCode("kr8", "std.prune("+config+"."+componentName+")")
		vm.ExtCode("kr8_unpruned", config+"."+componentName)

		if pruneFlag {
			input = "std.prune(import '" + args[0] + "')"
		} else {
			input = "( import '" + args[0] + "')"
		}
		j, err := vm.EvaluateSnippet("file", input)

		if err != nil {
			log.Panic("Error evaluating jsonnet snippet: ", err)
		}
		switch outputFormat {
		case "yaml":
			yaml, err := goyaml.JSONToYAML([]byte(j))
			if err != nil {
				log.Panic("Error converting JSON to YAML: ", err)
			}
			fmt.Println(string(yaml))
		case "stream": // output yaml stream
			var o []interface{}
			if err := json.Unmarshal([]byte(j), &o); err != nil {
				log.Panic(err)
			}
			for _, jobj := range o {
				fmt.Println("---")
				buf, err := goyaml.Marshal(jobj)
				if err != nil {
					log.Panic(err)
				}
				fmt.Println(string(buf))
			}
		case "json":
			formatted := Pretty(j, colorOutput)
			fmt.Println(formatted)
		default:
			log.Fatal("Output format must be json, yaml or stream")
		}
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
				log.Panic("Error decoding decoding yaml stream", err)
			}
			if len(bytes) == 0 {
				continue
			}
			jsondata, err := yaml.ToJSON(bytes)
			if err != nil {
				log.Panic("Error encoding yaml to JSON", err)
			}
			if string(jsondata) == "null" {
				// skip empty json
				continue
			}
			_, _, err = unstructured.UnstructuredJSONScheme.Decode(jsondata, nil, nil)
			if err != nil {
				log.Panic("Error handling unstructured JSON", err)
			}
			jsa = append(jsa, jsondata)
		}
		for _, j := range jsa {
			out, err := goyaml.JSONToYAML(j)
			if err != nil {
				log.Panic("Error encoding JSON to YAML", err)
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
