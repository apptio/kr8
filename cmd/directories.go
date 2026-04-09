package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
)

type componentDef struct {
	Path   string      `json:"path"`
	Enable interface{} `json:"enable,omitempty"` // Can be bool or will be evaluated as jsonnet expression
}

func (c *Clusters) addItem(item Cluster) Clusters {
	c.Cluster = append(c.Cluster, item)
	return *c
}

func getClusters(searchDir string) (Clusters, error) {

	fileList := make([]string, 0)

	e := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		fileList = append(fileList, path)
		return err
	})

	if e != nil {
		log.Fatal().Err(e).Msg("Error building cluster list: ")

	}

	ClusterData := []Cluster{}
	c := Clusters{ClusterData}

	for _, file := range fileList {

		splitFile := strings.Split(file, "/")
		// get the filename
		fileName := splitFile[len(splitFile)-1]

		if fileName == "cluster.jsonnet" {
			entry := Cluster{Name: splitFile[len(splitFile)-2], Path: strings.Join(splitFile[:len(splitFile)-1], "/")}
			c.addItem(entry)

		}
	}

	return c, nil

}

func getCluster(searchDir string, clusterName string) string {

	var clusterPath string

	e := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		dir, file := filepath.Split(path)
		if filepath.Base(dir) == clusterName && file == "cluster.jsonnet" {
			clusterPath = path
			return nil
		} else {
			return err
		}
	})

	if e != nil {
		log.Fatal().Err(e).Msg("Error building cluster list: ")

	}

	if clusterPath == "" {
		log.Fatal().Msg("Could not find cluster: " + clusterName)
	}

	return clusterPath

}

func getClusterParams(basePath string, targetPath string) []string {

	// a slice to store results
	var results []string
	results = append(results, targetPath)

	// remove the cluster.jsonnet
	splitFile := strings.Split(targetPath, "/")

	// gets the targetdir without the cluster.jsonnet
	targetDir := strings.Join(splitFile[:len(splitFile)-1], "/")

	// walk through the directory hierachy
	for {
		rel, _ := filepath.Rel(basePath, targetDir)

		// check if there's a params.json in the folder
		if _, err := os.Stat(targetDir + "/params.jsonnet"); err == nil {
			results = append(results, targetDir+"/params.jsonnet")
		}

		// stop if we're in the basePath
		if rel == "." {
			break
		}

		// next!
		targetDir += "/.."
	}

	// jsonnet's import order matters, so we need to reverse the slice
	last := len(results) - 1
	for i := 0; i < len(results)/2; i++ {
		results[i], results[last-i] = results[last-i], results[i]
	}

	return results
}

// only render cluster params (_cluster), without components
func renderClusterParamsOnly(cmd *cobra.Command, clusterName string, clusterParams string, prune bool) string {
	var params []string
	if clusterName != "" {
		clusterPath := getCluster(clusterDir, clusterName)
		params = getClusterParams(clusterDir, clusterPath)
	}
	if clusterParams != "" {
		params = append(params, clusterParams)
	}
	renderedParams := renderJsonnet(cmd, params, "._cluster", prune, "", "clusterparams")

	return renderedParams
}

// render cluster params, merged with one or more component's parameters. Empty componentName list renders all component parameters
func renderClusterParams(cmd *cobra.Command, clusterName string, componentNames []string, clusterParams string, prune bool) string {
	if clusterName == "" && clusterParams == "" {
		log.Fatal().Msg("Please specify a --cluster name and/or --clusterparams")
	}

	var params []string
	var componentMap map[string]componentDef

	if clusterName != "" {
		clusterPath := getCluster(clusterDir, clusterName)
		// prepend _components.jsonnet if it exists
		registryPath := baseDir + "/instances/_components.jsonnet"
		if _, err := os.Stat(registryPath); err == nil {
			log.Debug().Str("registry", registryPath).Str("targetPath", clusterPath).Msg("Loading component registry")
			params = append(params, registryPath)
		}
		// get list of jsonnet files that form the cluster's params
		params = append(params, getClusterParams(clusterDir, clusterPath)...)
	}
	if clusterParams != "" {
		params = append(params, clusterParams)
	}

	compParams := renderJsonnet(cmd, params, "", true, "", "clusterparams")

	compString := gjson.Get(compParams, "_components")
	err := json.Unmarshal([]byte(compString.String()), &componentMap)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse component map")
	}

	// Filter components based on enable field
	// If enable field is missing, default to true (backward compatible)
	// If enable is false, skip the component
	filteredComponentMap := make(map[string]componentDef)
	for key, value := range componentMap {
		enabled := true // default to enabled for backward compatibility

		if value.Enable != nil {
			// Check if it's a boolean
			if enableBool, ok := value.Enable.(bool); ok {
				enabled = enableBool
			} else {
				// If not a boolean, it should have been evaluated by jsonnet already
				// This shouldn't happen in normal cases, but log a warning
				log.Warn().Str("component", key).Msg("Enable field is not a boolean, defaulting to enabled")
			}
		}

		if enabled {
			filteredComponentMap[key] = value
		} else {
			log.Debug().Str("component", key).Msg("renderClusterParams: Component disabled by enable field")
		}
	}
	componentMap = filteredComponentMap

	// Build list of components to process
	// If componentNames is provided, use that list; otherwise use all components from componentMap
	componentsToProcess := make(map[string]componentDef)
	if len(componentNames) > 0 {
		// Only include specified components
		for _, key := range componentNames {
			if value, ok := componentMap[key]; ok {
				componentsToProcess[key] = value
			}
		}
	} else {
		// Use all components, but filter by componentName if set (legacy behavior)
		for key, value := range componentMap {
			if componentName != "" && key != componentName {
				continue
			}
			componentsToProcess[key] = value
		}
	}

	// Process each component
	componentDefaultsMerged := "{"
	for key, value := range componentsToProcess {
		// Load component params.jsonnet
		paramsPath := baseDir + "/" + value.Path + "/params.jsonnet"
		paramsContent, err := os.ReadFile(paramsPath)
		if err != nil {
			log.Fatal().Err(err).Msg("Error reading " + paramsPath)
		}

		// Determine component base name from path (e.g., "components/services/standard_service" -> "standard_service")
		componentBaseName := filepath.Base(value.Path)

		// Check for instance-specific params in global instances directory
		// First try: instances/<component_base_name>/<instance_name>.jsonnet
		// Second try: instances/<instance_name>.jsonnet
		var instancePath string
		var instanceContent []byte

		instancePathWithSubdir := baseDir + "/instances/" + componentBaseName + "/" + key + ".jsonnet"
		instanceContent, err = os.ReadFile(instancePathWithSubdir)
		if err == nil {
			instancePath = instancePathWithSubdir
		} else if os.IsNotExist(err) {
			// Try without subdirectory
			instancePathNoSubdir := baseDir + "/instances/" + key + ".jsonnet"
			instanceContent, err = os.ReadFile(instancePathNoSubdir)
			if err == nil {
				instancePath = instancePathNoSubdir
			}
		}

		if err == nil {
			// Instance file exists - merge params.jsonnet + instance.jsonnet
			log.Debug().Str("component", key).Str("instance", instancePath).Msg("Loading instance parameters")
			componentDefaultsMerged = componentDefaultsMerged + "'" + key + "'" + ": (" + string(paramsContent) + ") + (" + string(instanceContent) + "),"
		} else if os.IsNotExist(err) {
			// Instance file doesn't exist - use only params.jsonnet (fallback to existing behavior)
			componentDefaultsMerged = componentDefaultsMerged + "'" + key + "': " + string(paramsContent) + ","
		} else {
			// Other error reading instance file
			log.Fatal().Err(err).Msg("Error reading instance file")
		}
	}
	componentDefaultsMerged = componentDefaultsMerged + "}"

	// we replace _components with the filtered list
	componentMapJson, _ := json.Marshal(componentMap)

	// compParams is a json string
	compParams = renderJsonnet(cmd, params, "{ _components: "+string(componentMapJson)+"}", prune, componentDefaultsMerged, "componentparams")

	return compParams
}
