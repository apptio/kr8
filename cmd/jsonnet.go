package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	gyaml "github.com/ghodss/yaml"
	jsonnet "github.com/google/go-jsonnet"
	jsonnetAst "github.com/google/go-jsonnet/ast"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
	"io"
	"io/ioutil"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"path/filepath"
	"strings"
)

var (
	pruneFlag       bool
	outputFormat    string
	extVarFileFlag  []string
	jsonnetIncludes []string
)

// Create Jsonnet VM. Configure with env vars and command line flags
/*

This code is copied almost verbatim from the kubecfg project: https://github.com/ksonnet/kubecfg

Copyright 2018 ksonnet

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.

*/
func JsonnetVM(cmd *cobra.Command) (*jsonnet.VM, error) {
	vm := jsonnet.MakeVM()
	RegisterNativeFuncs(vm)

	flags := cmd.Flags()

	jpath := filepath.SplitList(os.Getenv("KR8_JPATH"))
	jpathArgs, err := flags.GetStringArray("jpath")
	if err != nil {
		return nil, err
	}
	jpath = append(jpath, jpathArgs...)

	vm.Importer(&jsonnet.FileImporter{
		JPaths: jpath,
	})

	extvarfiles, err := flags.GetStringSlice("ext-str-file")
	if err != nil {
		panic(err)
	}
	for _, extvar := range extvarfiles {
		kv := strings.SplitN(extvar, "=", 2)
		if len(kv) != 2 {
			log.Panic("Failed to parse %s: missing '=' in %s", "ext-str-file", extvar)
		}
		v, err := ioutil.ReadFile(kv[1])
		if err != nil {
			panic(err)
		}
		vm.ExtVar(kv[0], string(v))
	}
	return vm, nil
}

// Takes a list of jsonnet files and imports each one and mixes them with "+"
func renderJsonnet(cmd *cobra.Command, files []string, param string, prune bool, prepend string) string {

	// copy the slice so that we don't unitentionally modify the original
	jsonnetPaths := make([]string, len(files[:0]))
	copy(jsonnetPaths, files[:0])

	// range through the files
	for _, s := range files {
		jsonnetPaths = append(jsonnetPaths, fmt.Sprintf("(import '%s')", s))
	}

	// Create a JSonnet VM
	vm, err := JsonnetVM(cmd)
	if err != nil {
		log.Panic("Error creating jsonnet VM:", err)
	}

	// Join the slices into a jsonnet compat string. Prepend code from "prepend" variable, if set.
	var jsonnetImport string
	if prepend != "" {
		jsonnetImport = prepend + "+" + strings.Join(jsonnetPaths, "+")
	} else {
		jsonnetImport = strings.Join(jsonnetPaths, "+")
	}

	if param != "" {
		jsonnetImport = "(" + jsonnetImport + ")" + param
	}

	if prune {
		// wrap in std.prune, to remove nulls, empty arrays and hashes
		jsonnetImport = "std.prune(" + jsonnetImport + ")"
	}

	// render the jsonnet
	out, err := vm.EvaluateSnippet("file", jsonnetImport)

	if err != nil {
		log.Panic("Error evaluating jsonnet snippet: ", err)
	}

	return out

}

// Native Jsonnet funcs to add
/*

This code is copied almost verbatim from the kubecfg project: https://github.com/ksonnet/kubecfg

Copyright 2018 ksonnet

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.

*/
func RegisterNativeFuncs(vm *jsonnet.VM) {
	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "parseJson",
		Params: []jsonnetAst.Identifier{"json"},
		Func: func(args []interface{}) (res interface{}, err error) {
			data := []byte(args[0].(string))
			err = json.Unmarshal(data, &res)
			return
		},
	})

	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "parseYaml",
		Params: []jsonnetAst.Identifier{"yaml"},
		Func: func(args []interface{}) (res interface{}, err error) {
			ret := []interface{}{}
			data := []byte(args[0].(string))
			d := k8syaml.NewYAMLToJSONDecoder(bytes.NewReader(data))
			for {
				var doc interface{}
				if err := d.Decode(&doc); err != nil {
					if err == io.EOF {
						break
					}
					return nil, err
				}
				ret = append(ret, doc)
			}
			return ret, nil
		},
	})
}

var jsonnetCmd = &cobra.Command{
	Use:   "jsonnet",
	Short: "Jsonnet utilities",
	Long:  `Utility commands to process jsonnet`,
}

var renderCmd = &cobra.Command{
	Use:   "render file [file ...]",
	Short: "Render a jsonnet file",
	Long:  `Render a jsonnet file to JSON or YAML`,

	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if clusterName == "" {
			log.Fatal("Please specify a cluster name")
		}
		clusterPath := getCluster(base, clusterName)
		params := getClusterParams(base, clusterPath)

		// VM
		vm, _ := JsonnetVM(cmd)

		config := renderJsonnet(cmd, params, "", true, "")
		if componentName != "" {
			// lookup the configured path for this component
			componentPrefix := gjson.Get(config, "_components."+componentName+".path")
			if componentPrefix.String() == "" {
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

			prepend := "{" + componentName + ": " + string(filec) + "}"
			config = renderJsonnet(cmd, params, "", true, prepend)
		}

		var input string
		// pass component, _cluster and _components as extvars
		vm.ExtCode("kr8_cluster", config+"._cluster")
		vm.ExtCode("kr8_components", config+"._components")
		vm.ExtCode("kr8", config+"."+componentName)
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
			yaml, err := gyaml.JSONToYAML([]byte(j))
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
				buf, err := gyaml.Marshal(jobj)
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

func init() {
	RootCmd.AddCommand(jsonnetCmd)
	jsonnetCmd.AddCommand(renderCmd)
	renderCmd.PersistentFlags().BoolVarP(&pruneFlag, "prune", "", true, "Prune null and empty objects from rendered json")
	renderCmd.PersistentFlags().StringVarP(&clusterName, "cluster", "c", "", "cluster to render params for")
	renderCmd.PersistentFlags().StringVarP(&componentName, "component", "C", "", "component to render params for")
	renderCmd.PersistentFlags().StringVarP(&outputFormat, "format", "F", "json", "Output forma: json, yaml, stream")
}
