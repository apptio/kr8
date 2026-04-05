package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	goyaml "github.com/ghodss/yaml"
	jsonnet "github.com/google/go-jsonnet"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
)

type safeString struct {
	mu     sync.Mutex
	config string
}

var (
	components       string
	clusters         string
	generateDir      string
	clIncludes       string
	clExcludes       string
	allClusterParams map[string]string
	statsMode        string
	showProgress     bool
)

// Global timing statistics
type TimingStats struct {
	mu                    sync.Mutex
	totalDuration         time.Duration
	clusterParamsDuration time.Duration
	clusterCount          int
	componentCount        int
	includeCount          int
	totalJsonnetEval      time.Duration
	totalYamlMarshal      time.Duration
	totalFileIO           time.Duration
	totalRenderParams     time.Duration
	totalVMSetup          time.Duration
	totalDirOps           time.Duration
}

var globalTimingStats TimingStats

// Helper functions for stats mode
func statsEnabled() bool {
	return statsMode != ""
}

func statsSummaryOnly() bool {
	return statsMode == "summary"
}

func genProcessCluster(cmd *cobra.Command, clusterName string, p *ants.Pool) {
	var t1 time.Time
	var t2 time.Duration
	var clusterStart time.Time
	if statsEnabled() {
		clusterStart = time.Now()
	}
	
	log.Debug().Str("cluster", clusterName).Msg("Process cluster")

	var clusterComponentsUnfilt map[string]componentDef
	var params []string

	clusterPath := getCluster(clusterDir, clusterName)
	registryPath := baseDir + "/instances/_components.jsonnet"
	if _, err := os.Stat(registryPath); err == nil {
		log.Debug().Str("registry", registryPath).Str("targetPath", clusterPath).Msg("Loading component registry")
		params = append(params, registryPath)
		// get list of jsonnet files that form the cluster's params
	}
	// get list of components for cluster
	params = append(params, getClusterParams(clusterDir, getCluster(clusterDir, clusterName))...)
	//clusterComponents := gjson.Parse(renderJsonnet(cmd, params, "._components", true, "", "clustercomponents")).Map()
	err := json.Unmarshal([]byte(renderJsonnet(cmd, params, "._components", true, "", "clustercomponents")), &clusterComponentsUnfilt)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse component map")
	}

	// Filter components based on enable field
	// If enable field is missing, default to true (backward compatible)
	// If enable is false, skip the component
	clusterComponents := make(map[string]componentDef)
	for key, value := range clusterComponentsUnfilt {
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
			clusterComponents[key] = value
		} else {
			log.Debug().Str("component", key).Msg("Component disabled by enable field")
		}
	}

	// get kr8 settings for cluster
	kr8Spec := gjson.Parse(renderJsonnet(cmd, params, "._kr8_spec", false, "", "kr8_spec"))
	postProcessorFunction := kr8Spec.Get("postprocessor").String()

	var clGenerateDir string
	if generateDir == "" {
		clGenerateDir = kr8Spec.Get("generate_dir").String()
		if clGenerateDir == "" {
			log.Fatal().Msg("_kr8_spec.generate_dir must be set in parameters or passed as generate-dir flag")
		}
	} else {
		clGenerateDir = generateDir
	}
	if !strings.HasPrefix(clGenerateDir, "/") {
		// if generateDir does not start with /, then it goes in baseDir
		clGenerateDir = baseDir + "/" + clGenerateDir
	}
	clusterDir := clGenerateDir + "/" + clusterName

	// if this is true, we don't use the full file path to generate output file names
	generateShortNames := kr8Spec.Get("generate_short_names").Bool()

	// if this is true, we prune component parameters
	pruneParams := kr8Spec.Get("prune_params").Bool()

	// create generateDir
	var dirOpsStart time.Time
	if statsEnabled() {
		dirOpsStart = time.Now()
	}
	if _, err := os.Stat(clGenerateDir); os.IsNotExist(err) {
		err = os.MkdirAll(clGenerateDir, os.ModePerm)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
	}
	// create cluster dir
	if _, err := os.Stat(clusterDir); os.IsNotExist(err) {
		err = os.MkdirAll(clusterDir, os.ModePerm)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
	}

	// list of current generated components directories
	d, err := os.Open(clusterDir)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	defer d.Close()
	read_all_dirs := -1
	generatedCompList, err := d.Readdirnames(read_all_dirs)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	if statsEnabled() {
		dirOpsDuration := time.Since(dirOpsStart)
		if !statsSummaryOnly() {
			log.Debug().Str("function", "genProcessCluster").Str("cluster", clusterName).Dur("duration", dirOpsDuration).Msg("STATS: Directory operations")
		}
		globalTimingStats.mu.Lock()
		globalTimingStats.totalDirOps += dirOpsDuration
		globalTimingStats.mu.Unlock()
	}

	// determine list of components to process
	var compList []string
	var currentCompList []string

	if components != "" {
		// only process specified component if it's defined in the cluster
		for _, b := range strings.Split(components, ",") {
			for _, c := range generatedCompList {
				matched, _ := regexp.MatchString("^"+b+"$", c)
				if matched {
					currentCompList = append(currentCompList, c)
				}
			}
			for c, _ := range clusterComponents {
				matched, _ := regexp.MatchString("^"+b+"$", c)
				if matched {
					compList = append(compList, c)
				}
			}
		}
	} else {
		for c, _ := range clusterComponents {
			compList = append(compList, c)
		}
		currentCompList = generatedCompList
	}
	sort.Strings(compList) // process components in sorted order

	// Sort out orphaned generated components directories
	tmpMap := make(map[string]struct{}, len(clusterComponents))
	for e, _ := range clusterComponents {
		tmpMap[e] = struct{}{}
	}

	for _, e := range currentCompList {
		if _, found := tmpMap[e]; !found {
			delcomp := filepath.Join(clusterDir, e)
			os.RemoveAll(delcomp)
			log.Info().Str("cluster", clusterName).
				Str("component", e).
				Msg("Deleting generated for component")
		}
	}

	if len(compList) == 0 { // this needs to be moved so purging above works first
		return
	}

	// render full params for cluster for all selected components
	if statsEnabled() {
		t1 = time.Now()
	}
	config := renderClusterParams(cmd, clusterName, compList, clusterParams, false)
	if statsEnabled() {
		t2 = time.Since(t1)
		if !statsSummaryOnly() {
			log.Debug().Str("function", "genProcessCluster").Str("cluster", clusterName).Dur("duration", t2).Msg("STATS: Render cluster params")
		}
		// Add to global stats
		globalTimingStats.mu.Lock()
		globalTimingStats.totalRenderParams += t2
		globalTimingStats.mu.Unlock()
	}
	
	// Parse config once to avoid repeated parsing in each component
	configParsed := gjson.Parse(config)
	
	// Extract and prune cluster config once - it's the same for all components
	// We need to evaluate std.prune() once to match the original behavior
	vm, _ := JsonnetVM(cmd)
	clusterConfigRaw := configParsed.Get("_cluster").Raw
	clusterConfig, err := vm.EvaluateAnonymousSnippet("prune_cluster", "std.prune("+clusterConfigRaw+")")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to prune cluster config")
	}

	var allconfig safeString

	var wg sync.WaitGroup
	//p, _ := ants.NewPool(4)
	for _, componentName := range compList {
		wg.Add(1)
		cName := componentName
		_ = p.Submit(func() {
			defer wg.Done()
			genProcessComponent(cmd, clusterName, cName, clusterDir, clGenerateDir, config, configParsed, clusterConfig, &allconfig, postProcessorFunction, pruneParams, generateShortNames)
		})
	}
	wg.Wait()
	
	if statsEnabled() {
		clusterDuration := time.Since(clusterStart)
		globalTimingStats.mu.Lock()
		globalTimingStats.clusterCount++
		globalTimingStats.componentCount += len(compList)
		globalTimingStats.mu.Unlock()
		
		if !statsSummaryOnly() {
			log.Info().Str("function", "genProcessCluster").Str("cluster", clusterName).
				Dur("total_duration", clusterDuration).
				Int("components", len(compList)).
				Msgf("STATS: Cluster total: %v (%d components)", clusterDuration, len(compList))
		}
	}
}

func genProcessComponent(cmd *cobra.Command, clusterName string, componentName string, clusterDir string, clGenerateDir string, config string, configParsed gjson.Result, clusterConfig string, allconfig *safeString, postProcessorFunction string, pruneParams bool, generateShortNames bool) {
	var t1 time.Time
	var t2 time.Duration
	var componentStart time.Time
	if statsEnabled() {
		componentStart = time.Now()
	}

	if showProgress {
		log.Info().Str("cluster", clusterName).
			Str("component", componentName).
			Msg("Process component")
	} else {
		log.Debug().Str("cluster", clusterName).
			Str("component", componentName).
			Msg("Process component")
	}

	// get kr8_spec from component's params - use pre-parsed config
	if statsEnabled() {
		t1 = time.Now()
	}
	spec := configParsed.Get(componentName + ".kr8_spec").Map()
	compPath := configParsed.Get("_components." + componentName + ".path").String()
	
	// Extract component params once - avoids jsonnet parsing "config.componentName" expression
	componentParams := configParsed.Get(componentName).Raw
	
	if statsEnabled() {
		t2 = time.Since(t1)
		if !statsSummaryOnly() {
			log.Debug().Str("function", "genProcessComponent").Str("component", componentName).Dur("duration", t2).Msg("STATS: Get spec and path")
		}
	}

	// spec is missing?
	if len(spec) == 0 {
		log.Fatal().Str("cluster", clusterName).
			Str("component", componentName).
			Msg("Component has no kr8_spec")
		return
	}

	// it's faster to create this VM for each component, rather than re-use
	var vmSetupStart time.Time
	if statsEnabled() {
		t1 = time.Now()
		vmSetupStart = time.Now()
	}
	vm, _ := JsonnetVM(cmd)
	if statsEnabled() {
		t2 = time.Since(t1)
		if !statsSummaryOnly() {
			log.Debug().Str("function", "genProcessComponent").Str("component", componentName).Dur("duration", t2).Msg("STATS: Create JsonnetVM")
		}
	}
	vm.ExtCode("kr8_cluster", clusterConfig)
	//vm.ExtCode("kr8_components", "std.prune("+config+"._components)")
	if postProcessorFunction != "" {
		vm.ExtCode("process", postProcessorFunction)
	} else {
		// default postprocessor just copies input
		vm.ExtCode("process", "function(input) input")
	}

	// prune params if required
	// Use pre-extracted component params to avoid jsonnet parsing the expression
	if pruneParams {
		vm.ExtCode("kr8", "std.prune("+componentParams+")")
	} else {
		vm.ExtCode("kr8", componentParams)
	}

	// add kr8_allparams extcode with all component params in the cluster
	if spec["enable_kr8_allparams"].Bool() {
		if statsEnabled() {
			t1 = time.Now()
		}
		// include full render of all component params
		allconfig.mu.Lock()
		if allconfig.config == "" {
			// only do this if we have not already cached it and don't already have it stored
			if components == "" {
				// all component params are in config
				allconfig.config = config
			} else {
				allconfig.config = renderClusterParams(cmd, clusterName, []string{}, clusterParams, false)
			}
		}
		vm.ExtCode("kr8_allparams", allconfig.config)
		allconfig.mu.Unlock()
		if statsEnabled() {
			t2 = time.Since(t1)
			if !statsSummaryOnly() {
				log.Debug().Str("function", "genProcessComponent").Str("component", componentName).Dur("duration", t2).Msg("STATS: Setup kr8_allparams")
			}
		}
	}

	// add kr8_allclusters extcode with every cluster's cluster level params
	if spec["enable_kr8_allclusters"].Bool() {
		// combine all the cluster params into a single object indexed by cluster name
		var allClusterParamsObject string
		allClusterParamsObject = "{ "
		for cl, clp := range allClusterParams {
			allClusterParamsObject = allClusterParamsObject + "'" + cl + "': " + clp + ","

		}
		allClusterParamsObject = allClusterParamsObject + "}"
		vm.ExtCode("kr8_allclusters", allClusterParamsObject)
	}

	// jpath always includes base lib. Add jpaths from spec if set
	jpath := []string{baseDir + "/lib"}
	for _, j := range spec["jpaths"].Array() {
		jpath = append(jpath, baseDir+"/"+compPath+"/"+j.String())
	}
	vm.Importer(&jsonnet.FileImporter{
		JPaths: jpath,
	})

	// file imports
	if statsEnabled() && len(spec["extfiles"].Map()) > 0 {
		t1 = time.Now()
	}
	for k, v := range spec["extfiles"].Map() {
		vpath := baseDir + "/" + compPath + "/" + v.String() // use full path for file
		extfile, err := os.ReadFile(vpath)
		if err != nil {
			log.Fatal().Err(err).Msg("Error importing extfile")
		}
		log.Debug().Str("cluster", clusterName).
			Str("component", componentName).
			Msg("Extfile: " + k + "=" + v.String())
		vm.ExtVar(k, string(extfile))
	}
	if statsEnabled() && len(spec["extfiles"].Map()) > 0 {
		t2 = time.Since(t1)
		if !statsSummaryOnly() {
			log.Debug().Str("function", "genProcessComponent").Str("component", componentName).Dur("duration", t2).Int("count", len(spec["extfiles"].Map())).Msg("STATS: Load extfiles")
		}
	}
	
	// Complete VM setup timing
	if statsEnabled() {
		vmSetupDuration := time.Since(vmSetupStart)
		if !statsSummaryOnly() {
			log.Debug().Str("function", "genProcessComponent").Str("component", componentName).Dur("duration", vmSetupDuration).Msg("STATS: Total VM setup")
		}
		globalTimingStats.mu.Lock()
		globalTimingStats.totalVMSetup += vmSetupDuration
		globalTimingStats.mu.Unlock()
	}

	componentDir := clusterDir + "/" + componentName
	// create component dir if needed
	if _, err := os.Stat(componentDir); os.IsNotExist(err) {
		err := os.MkdirAll(componentDir, os.ModePerm)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
	}

	outputFileMap := make(map[string]bool)
	
	var totalJsonnetEval, totalYamlMarshal, totalFileIO time.Duration
	includeCount := len(spec["includes"].Array())
	
	// generate each included file
	for _, include := range spec["includes"].Array() {
		var filename string
		var outputDir string
		var genFileNamePrefix string

		itype := include.Type.String()
		outputDir = componentDir
		if itype == "String" {
			// include is just a string for the filename
			filename = include.String()
		} else if itype == "JSON" {
			// include is a map with multiple fields
			inc_spec := include.Map()
			filename = inc_spec["file"].String()
			if inc_spec["dest_dir"].Exists() {
				// handle alternate output directory for file
				altdir := inc_spec["dest_dir"].String()
				// dir is always relative to generate dir
				outputDir = clGenerateDir + "/" + altdir
				// ensure this directory exists
				if _, err := os.Stat(outputDir); os.IsNotExist(err) {
					err = os.MkdirAll(outputDir, os.ModePerm)
					if err != nil {
						log.Fatal().Err(err).Msg("")
					}
				}
			}
			if inc_spec["dest_name"].Exists() {
				// override destination file name
				genFileNamePrefix = inc_spec["dest_name"].String()
			}
		}
		file_extension := filepath.Ext(filename)
		if genFileNamePrefix == "" {
			if generateShortNames {
				sbase := filepath.Base(filename)
				genFileNamePrefix = sbase[0 : len(sbase)-len(file_extension)]
			} else {
				// replaces slashes with _ in multi-dir paths and replace extension with yaml
				genFileNamePrefix = strings.ReplaceAll(filename[0:len(filename)-len(file_extension)], "/", "_")
			}
		}
		outputFile := outputDir + "/" + genFileNamePrefix + ".yaml"
		// remember output filename for purging files
		outputFileMap[genFileNamePrefix+".yaml"] = true

		log.Debug().Str("cluster", clusterName).
			Str("component", componentName).
			Msg("Process file: " + filename + " -> " + outputFile)

		var input string
		switch file_extension {
		case ".jsonnet", ".json":
			// file is processed as an ExtCode input, so that we can postprocess it
			// in the snippet
			input = "( import '" + baseDir + "/" + compPath + "/" + filename + "')"
		case ".yaml":
			input = "std.native('parseYaml')(importstr '" + baseDir + "/" + compPath + "/" + filename + "')"
		default:
			log.Fatal().Str("cluster", clusterName).
				Str("component", componentName).
				Str("file", filename).
				Msg("Unsupported file extension")
		}

		vm.ExtCode("input", input)

		// Pass in an external var containing some metadata about this include. Currently just the dest_name prefix.
		vm.ExtCode("kr8_include_meta", "{\"dest_name\":\""+genFileNamePrefix+"\"}")

		if statsEnabled() {
			t1 = time.Now()
		}
		j, err := vm.EvaluateAnonymousSnippet(include.String(), "std.extVar('process')(std.extVar('input'))")
		if statsEnabled() {
			t2 = time.Since(t1)
			totalJsonnetEval += t2
			if !statsSummaryOnly() {
				log.Debug().Str("function", "genProcessComponent").Str("component", componentName).Str("include", include.String()).Dur("duration", t2).Msg("STATS: Evaluate jsonnet for include")
			}
		}
		if err != nil {
			log.Fatal().Str("cluster", clusterName).
				Str("component", componentName).
				Str("file", filename).Err(err).Msg("Error evaluating jsonnet snippet")
		}

		// create output file contents in a string first, as a yaml stream
		if statsEnabled() {
			t1 = time.Now()
		}
		var o []interface{}
		var outStr string
		if err := json.Unmarshal([]byte(j), &o); err != nil {
			log.Fatal().Err(err).Msg("")
		}
		for _, jobj := range o {
			outStr = outStr + "---\n"
			buf, err := goyaml.Marshal(jobj)
			if err != nil {
				log.Fatal().Err(err).Msg("")
			}
			outStr = outStr + string(buf) + "\n"
		}
		if statsEnabled() {
			t2 = time.Since(t1)
			totalYamlMarshal += t2
		}

		// only write file if it does not exist, or the generated contents does not match what is on disk
		if statsEnabled() {
			t1 = time.Now()
		}
		var updateNeeded bool
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			log.Debug().Str("cluster", clusterName).
				Str("component", componentName).
				Msg("Creating " + outputFile)
			updateNeeded = true
		} else {
			currentContents, err := os.ReadFile(outputFile)
			if err != nil {
				log.Fatal().Err(err).Msg("Error reading file")
			}
			if string(currentContents) != outStr {
				updateNeeded = true
				log.Debug().Str("cluster", clusterName).
					Str("component", componentName).
					Msg("Updating: " + outputFile)
			}
		}
		if updateNeeded {
			f, err := os.Create(outputFile)
			if err != nil {
				log.Fatal().Err(err).Msg("")
			}
			defer f.Close()
			_, err = f.WriteString(outStr)
			if err != nil {
				log.Fatal().Err(err).Msg("")
			}

			f.Close()
		}
		if statsEnabled() {
			t2 = time.Since(t1)
			totalFileIO += t2
		}
	}
	
	if statsEnabled() && includeCount > 0 {
		avgJsonnet := totalJsonnetEval / time.Duration(includeCount)
		avgYaml := totalYamlMarshal / time.Duration(includeCount)
		avgIO := totalFileIO / time.Duration(includeCount)
		if !statsSummaryOnly() {
			log.Debug().Str("function", "genProcessComponent").Str("component", componentName).
				Dur("total_jsonnet", totalJsonnetEval).
				Dur("total_yaml", totalYamlMarshal).
				Dur("total_io", totalFileIO).
				Dur("avg_jsonnet", avgJsonnet).
				Dur("avg_yaml", avgYaml).
				Dur("avg_io", avgIO).
				Int("includes", includeCount).
				Msg("STATS: Process includes summary")
		}
	}
	
	// purge any yaml files in the output dir that were not generated
	if !spec["disable_output_clean"].Bool() {
		if statsEnabled() {
			t1 = time.Now()
		}
		// clean component dir
		d, err := os.Open(componentDir)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
		defer d.Close()
		names, err := d.Readdirnames(-1)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
		for _, name := range names {
			if _, ok := outputFileMap[name]; ok {
				// file is managed
				continue
			}
			if filepath.Ext(name) == ".yaml" {
				delfile := filepath.Join(componentDir, name)
				err = os.RemoveAll(delfile)
				if err != nil {
					log.Fatal().Err(err).Msg("")
				}
				log.Debug().Str("cluster", clusterName).
					Str("component", componentName).
					Msg("Deleted: " + delfile)
			}
		}
		d.Close()
		if statsEnabled() {
			t2 = time.Since(t1)
			if !statsSummaryOnly() {
				log.Debug().Str("function", "genProcessComponent").Str("component", componentName).Dur("duration", t2).Msg("STATS: Clean output directory")
			}
		}
	}
	
	if statsEnabled() {
		componentDuration := time.Since(componentStart)
		
		// Update global stats
		globalTimingStats.mu.Lock()
		globalTimingStats.includeCount += includeCount
		globalTimingStats.totalJsonnetEval += totalJsonnetEval
		globalTimingStats.totalYamlMarshal += totalYamlMarshal
		globalTimingStats.totalFileIO += totalFileIO
		globalTimingStats.mu.Unlock()
		
		if !statsSummaryOnly() {
			log.Info().Str("function", "genProcessComponent").Str("component", componentName).
				Dur("total_duration", componentDuration).
				Dur("jsonnet_eval_total", totalJsonnetEval).
				Dur("yaml_marshal_total", totalYamlMarshal).
				Dur("file_io_total", totalFileIO).
				Int("includes", includeCount).
				Msgf("STATS: Component total: %v (jsonnet: %v, yaml: %v, io: %v)",
					componentDuration, totalJsonnetEval, totalYamlMarshal, totalFileIO)
		}
	}
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate components",
	Long:  `Generate components in clusters`,

	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		var generateStart time.Time
		if statsEnabled() {
			generateStart = time.Now()
		}

		var clusterList []string

		// get list of all clusters, render cluster level params for all of them in parallel
		allClusterParams = make(map[string]string)
		allClusters, err := getClusters(clusterDir)
		if err != nil {
			log.Fatal().Err(err).Msg("Error getting list of clusters")
		}
		
		// Use a mutex to safely write to allClusterParams map
		var mu sync.Mutex
		var wgParams sync.WaitGroup
		
		var t1 time.Time
		if statsEnabled() {
			t1 = time.Now()
		}
		
		for _, c := range allClusters.Cluster {
			wgParams.Add(1)
			clusterName := c.Name
			go func() {
				defer wgParams.Done()
				params := renderClusterParamsOnly(cmd, clusterName, "", false)
				mu.Lock()
				allClusterParams[clusterName] = params
				mu.Unlock()
			}()
		}
		wgParams.Wait()
		
		if statsEnabled() {
			clusterParamsDuration := time.Since(t1)
			globalTimingStats.mu.Lock()
			globalTimingStats.clusterParamsDuration = clusterParamsDuration
			globalTimingStats.mu.Unlock()
			if !statsSummaryOnly() {
				log.Info().Str("function", "generate.Run").
					Dur("duration", clusterParamsDuration).
					Int("clusters", len(allClusters.Cluster)).
					Msg("STATS: Rendered all cluster params in parallel")
			}
		}

		for c, _ := range allClusterParams {
			if clIncludes != "" || clExcludes != "" {
				gjresult := gjson.Parse(allClusterParams[c])
				// includes
				if clIncludes != "" {
					// filter on cluster parameters, passed in gjson path notation with either
					// "=" for equality or "~" for regex match
					var include bool
					for _, b := range strings.Split(clIncludes, ",") {
						include = false
						// equality match
						kv := strings.SplitN(b, "=", 2)
						if len(kv) == 2 {
							if gjresult.Get(kv[0]).String() == kv[1] {
								include = true
							}
						} else {
							// regex match
							kv := strings.SplitN(b, "~", 2)
							if len(kv) == 2 {
								matched, _ := regexp.MatchString(kv[1], gjresult.Get(kv[0]).String())
								if matched {
									include = true
								}
							}
						}
						if !include {
							break
						}
					}
					if !include {
						continue
					}
				}
				// excludes
				if clExcludes != "" {
					// filter on cluster parameters, passed in gjson path notation with either
					// "=" for equality or "~" for regex match
					var exclude bool
					for _, b := range strings.Split(clExcludes, ",") {
						exclude = false
						// equality match
						kv := strings.SplitN(b, "=", 2)
						if len(kv) == 2 {
							if gjresult.Get(kv[0]).String() == kv[1] {
								exclude = true
							}
						} else {
							// regex match
							kv := strings.SplitN(b, "~", 2)
							if len(kv) == 2 {
								matched, _ := regexp.MatchString(kv[1], gjresult.Get(kv[0]).String())
								if matched {
									exclude = true
								}
							}
						}
						if exclude {
							break
						}
					}
					if exclude {
						continue
					}
				}
			}

			if clusters == "" {
				// all clusters
				clusterList = append(clusterList, c)
			} else {
				// match --clusters list
				for _, b := range strings.Split(clusters, ",") {
					// match cluster names as anchored regex
					matched, _ := regexp.MatchString("^"+b+"$", c)
					if matched {
						clusterList = append(clusterList, c)
						break
					}
				}

			}
		}

		var wg sync.WaitGroup
		parallel, err := cmd.Flags().GetInt("parallel")
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
		log.Debug().Msg("Parallel set to " + strconv.Itoa(parallel))

		ants_cp, _ := ants.NewPool(parallel)
		ants_cl, _ := ants.NewPool(parallel)

		for _, clusterName := range clusterList {
			wg.Add(1)
			cl := clusterName
			_ = ants_cl.Submit(func() {
				defer wg.Done()
				genProcessCluster(cmd, cl, ants_cp)
			})
		}
		wg.Wait()
		
		// Print final stats summary
		if statsEnabled() {
			totalDuration := time.Since(generateStart)
			globalTimingStats.mu.Lock()
			globalTimingStats.totalDuration = totalDuration
			globalTimingStats.mu.Unlock()
			
			// Helper function to format duration to 2 decimal places in seconds
			formatDuration := func(d time.Duration) string {
				return fmt.Sprintf("%.2fs", d.Seconds())
			}
			
			// Calculate total CPU time (sum of all parallel operations)
			totalCPUTime := globalTimingStats.clusterParamsDuration +
				globalTimingStats.totalRenderParams +
				globalTimingStats.totalVMSetup +
				globalTimingStats.totalJsonnetEval +
				globalTimingStats.totalYamlMarshal +
				globalTimingStats.totalFileIO +
				globalTimingStats.totalDirOps
			
			// Calculate percentages based on CPU time (not wall-clock time)
			cpuMs := float64(totalCPUTime.Milliseconds())
			clusterParamsPct := int(float64(globalTimingStats.clusterParamsDuration.Milliseconds()) / cpuMs * 100)
			renderParamsPct := int(float64(globalTimingStats.totalRenderParams.Milliseconds()) / cpuMs * 100)
			vmSetupPct := int(float64(globalTimingStats.totalVMSetup.Milliseconds()) / cpuMs * 100)
			jsonnetEvalPct := int(float64(globalTimingStats.totalJsonnetEval.Milliseconds()) / cpuMs * 100)
			yamlMarshalPct := int(float64(globalTimingStats.totalYamlMarshal.Milliseconds()) / cpuMs * 100)
			fileIOPct := int(float64(globalTimingStats.totalFileIO.Milliseconds()) / cpuMs * 100)
			dirOpsPct := int(float64(globalTimingStats.totalDirOps.Milliseconds()) / cpuMs * 100)
			
			// Calculate parallelism factor
			parallelismFactor := float64(totalCPUTime.Milliseconds()) / float64(totalDuration.Milliseconds())
			
			// Format summary message with 2 decimal places - counts first, then timings
			summaryMsg := fmt.Sprintf("STATS SUMMARY: Clusters=%d Components=%d Includes=%d | Wall=%s CPU=%s Parallel=%.1fx | ClusterParams=%s(%d%%) RenderParams=%s(%d%%) VMSetup=%s(%d%%) JsonnetEval=%s(%d%%) YAMLMarshal=%s(%d%%) FileIO=%s(%d%%) DirOps=%s(%d%%)",
				globalTimingStats.clusterCount,
				globalTimingStats.componentCount,
				globalTimingStats.includeCount,
				formatDuration(totalDuration),
				formatDuration(totalCPUTime),
				parallelismFactor,
				formatDuration(globalTimingStats.clusterParamsDuration), clusterParamsPct,
				formatDuration(globalTimingStats.totalRenderParams), renderParamsPct,
				formatDuration(globalTimingStats.totalVMSetup), vmSetupPct,
				formatDuration(globalTimingStats.totalJsonnetEval), jsonnetEvalPct,
				formatDuration(globalTimingStats.totalYamlMarshal), yamlMarshalPct,
				formatDuration(globalTimingStats.totalFileIO), fileIOPct,
				formatDuration(globalTimingStats.totalDirOps), dirOpsPct)
			
			// Print summary without structured logging fields
			log.Info().Msg(summaryMsg)
		}
	},
}

func init() {
	RootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVarP(&clusterParams, "clusterparams", "", "", "provide cluster params as single file - can be combined with --cluster to override cluster")
	generateCmd.Flags().StringVarP(&clusters, "clusters", "", "", "clusters to generate - comma separated list of cluster names and/or regular expressions ")
	generateCmd.Flags().StringVarP(&components, "components", "", "", "components to generate - comma separated list of component names and/or regular expressions")
	generateCmd.Flags().StringVarP(&generateDir, "generate-dir", "", "", "output directory")
	generateCmd.Flags().StringVarP(&clIncludes, "clincludes", "", "", "filter included cluster by including clusters with matching cluster parameters - comma separate list of key/value conditions separated by = or ~ (for regex match)")
	generateCmd.Flags().StringVarP(&clExcludes, "clexcludes", "", "", "filter included cluster by excluding clusters with matching cluster parameters - comma separate list of key/value conditions separated by = or ~ (for regex match)")
	generateCmd.Flags().IntP("parallel", "", runtime.GOMAXPROCS(0), "parallelism - defaults to GOMAXPROCS")
	generateCmd.Flags().StringVarP(&statsMode, "stats", "", "summary", "enable statistics: empty/all for detailed logs, 'summary' for summary only")
	generateCmd.Flags().BoolVarP(&showProgress, "progress", "", true, "show progress messages (Process cluster/component)")
	viper.BindPFlag("clincludes", generateCmd.PersistentFlags().Lookup("clincludes"))
	viper.BindPFlag("clexcludes", generateCmd.PersistentFlags().Lookup("clexcludes"))
}
