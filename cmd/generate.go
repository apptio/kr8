package cmd

import (
	"encoding/json"
	goyaml "github.com/ghodss/yaml"
	jsonnet "github.com/google/go-jsonnet"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	allClusterParams map[string]string
)

func genProcessCluster(cmd *cobra.Command, clusterName string, p *ants.Pool) {
	log.Debug().Str("cluster", clusterName).Msg("Process cluster")

	// get list of components for cluster
	params := getClusterParams(clusterDir, getCluster(clusterDir, clusterName))
	clusterComponents := gjson.Parse(renderJsonnet(cmd, params, "._components", true, "", "clustercomponents")).Map()

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

	// determine list of components to process
	var compList []string
	if components != "" {
		// only process specified component if it's defined in the cluster
		for c, _ := range clusterComponents {
			for _, b := range strings.Split(components, ",") {
				matched, _ := regexp.MatchString("^"+b+"$", c)
				if matched {
					compList = append(compList, c)
				}
			}
		}
		if len(compList) == 0 {
			return
		}
	} else {
		for c, _ := range clusterComponents {
			compList = append(compList, c)
		}
	}
	sort.Strings(compList) // process components in sorted order

	// render full params for cluster for all selected components
	config := renderClusterParams(cmd, clusterName, compList, clusterParams, false)

	var allconfig safeString

	var wg sync.WaitGroup
	//p, _ := ants.NewPool(4)
	for _, componentName := range compList {
		wg.Add(1)
		cName := componentName
		_ = p.Submit(func() {
			defer wg.Done()
			genProcessComponent(cmd, clusterName, cName, clusterDir, clGenerateDir, config, &allconfig, postProcessorFunction, pruneParams, generateShortNames)
		})
	}
	wg.Wait()

}

func genProcessComponent(cmd *cobra.Command, clusterName string, componentName string, clusterDir string, clGenerateDir string, config string, allconfig *safeString, postProcessorFunction string, pruneParams bool, generateShortNames bool) {

	log.Info().Str("cluster", clusterName).
		Str("component", componentName).
		Msg("Process component")

	// get kr8_spec from component's params
	spec := gjson.Get(config, componentName+".kr8_spec").Map()
	compPath := gjson.Get(config, "_components."+componentName+".path").String()

	// spec is missing?
	if len(spec) == 0 {
		log.Fatal().Str("cluster", clusterName).
			Str("component", componentName).
			Msg("Component has no kr8_spec")
		return
	}

	// it's faster to create this VM for each component, rather than re-use
	vm, _ := JsonnetVM(cmd)
	vm.ExtCode("kr8_cluster", "std.prune("+config+"._cluster)")
	//vm.ExtCode("kr8_components", "std.prune("+config+"._components)")
	if postProcessorFunction != "" {
		vm.ExtCode("process", postProcessorFunction)
	} else {
		// default postprocessor just copies input
		vm.ExtCode("process", "function(input) input")
	}

	// prune params if required
	if pruneParams {
		vm.ExtCode("kr8", "std.prune("+config+"."+componentName+")")
	} else {
		vm.ExtCode("kr8", config+"."+componentName)
	}

	// add kr8_allparams extcode with all component params in the cluster
	if spec["enable_kr8_allparams"].Bool() {
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
	for k, v := range spec["extfiles"].Map() {
		vpath := baseDir + "/" + compPath + "/" + v.String() // use full path for file
		extfile, err := ioutil.ReadFile(vpath)
		if err != nil {
			log.Fatal().Err(err).Msg("Error importing extfile")
		}
		log.Debug().Str("cluster", clusterName).
			Str("component", componentName).
			Msg("Extfile: " + k + "=" + v.String())
		vm.ExtVar(k, string(extfile))
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
	// generate each included file
	for _, include := range spec["includes"].Array() {
		var filename string
		var outputDir string
		var sfile string

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
				sfile = inc_spec["dest_name"].String()
			}
		}
		file_extension := filepath.Ext(filename)
		if sfile == "" {
			if generateShortNames {
				sbase := filepath.Base(filename)
				sfile = sbase[0 : len(sbase)-len(file_extension)]
			} else {
				// replaces slashes with _ in multi-dir paths and replace extension with yaml
				sfile = strings.ReplaceAll(filename[0:len(filename)-len(file_extension)], "/", "_")
			}
		}
		outputFile := outputDir + "/" + sfile + ".yaml"
		// remember output filename for purging files
		outputFileMap[sfile+".yaml"] = true

		log.Debug().Str("cluster", clusterName).
			Str("component", componentName).
			Msg("Process file: " + filename + " -> " + outputFile)

		var input string
		switch file_extension {
		case ".jsonnet":
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
		j, err := vm.EvaluateAnonymousSnippet(include.String(), "std.extVar('process')(std.extVar('input'))")
		if err != nil {
			log.Fatal().Str("cluster", clusterName).
				Str("component", componentName).
				Str("file", filename).Err(err).Msg("Error evaluating jsonnet snippet")
		}

		// create output file contents in a string first, as a yaml stream
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

		// only write file if it does not exist, or the generated contents does not match what is on disk
		var updateNeeded bool
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			log.Debug().Str("cluster", clusterName).
				Str("component", componentName).
				Msg("Creating " + outputFile)
			updateNeeded = true
		} else {
			currentContents, err := ioutil.ReadFile(outputFile)
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
	}
	// purge any yaml files in the output dir that were not generated
	if !spec["disable_output_clean"].Bool() {
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
	}
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate components",
	Long:  `Generate components in clusters`,

	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {

		var clusterList []string

		// get list of all clusters, render cluster level params for all of them
		allClusterParams = make(map[string]string)
		allClusters, err := getClusters(clusterDir)
		if err != nil {
			log.Fatal().Err(err).Msg("Error getting list of clusters")
		}
		for _, c := range allClusters.Cluster {
			allClusterParams[c.Name] = renderClusterParamsOnly(cmd, c.Name, "", false)
		}

		for c, _ := range allClusterParams {
			if clIncludes != "" {
				gjresult := gjson.Parse(allClusterParams[c])
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
	},
}

func init() {
	RootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVarP(&clusterParams, "clusterparams", "", "", "provide cluster params as single file - can be combined with --cluster to override cluster")
	generateCmd.Flags().StringVarP(&clusters, "clusters", "", "", "clusters to generate - comma separated list of cluster names and/or regular expressions ")
	generateCmd.Flags().StringVarP(&components, "components", "", "", "components to generate - comma separated list of component names and/or regular expressions")
	generateCmd.Flags().StringVarP(&generateDir, "generate-dir", "", "", "output directory")
	generateCmd.Flags().StringVarP(&clIncludes, "clincludes", "", "", "filter included cluster by matching cluster parameters - comma separate list of key/value conditions separated by = or ~ (for regex match)")
	generateCmd.Flags().IntP("parallel", "", runtime.GOMAXPROCS(0), "parallelism - defaults to GOMAXPROCS")
}
