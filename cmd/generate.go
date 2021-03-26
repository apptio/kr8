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
	"sort"
	"strings"
	"sync"
)

var (
	components  string
	clusters    string
	generateDir string
)

func genProcessCluster(cmd *cobra.Command, clusterName string) {
	log.Debug().Str("cluster", clusterName).Msg("Process cluster")

	// get list of components for cluster
	params := getClusterParams(clusterDir, getCluster(clusterDir, clusterName))
	clusterComponents := gjson.Parse(renderJsonnet(cmd, params, "._components", false, "", "clustercomponents")).Map()

	// get kr8 settings for cluster
	kr8Spec := gjson.Parse(renderJsonnet(cmd, params, "._kr8_spec", false, "","kr8_spec"))
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
		err = os.Mkdir(clGenerateDir, os.ModePerm)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
	}
	// create cluster dir
	if _, err := os.Stat(clusterDir); os.IsNotExist(err) {
		err = os.Mkdir(clusterDir, os.ModePerm)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
	}

	// determine list of components to process
	var compList []string
	if components != "" {
		// only process specified component if it's defined in the cluster
		for _, c := range strings.Split(components, ",") {
			if _, ok := clusterComponents[c]; ok {
				compList = append(compList, c)
			}
		}
		if len(compList) == 0 {
			return
		}
	} else {
		// get list of all components in cluster
		// FIXME: add filtering here
		for c, _ := range clusterComponents {
			compList = append(compList, c)
		}
	}
	sort.Strings(compList) // process components in sorted order

	// render full params for cluster for all selected components
	config := renderClusterParams(cmd, clusterName, compList, clusterParams, false)

	var allconfig string

	for _, componentName := range compList {

		log.Info().Str("cluster", clusterName).
			Str("component", componentName).
			Msg("Process component")

		// get kr8_spec from component's params
		spec := gjson.Get(config, componentName+".kr8_spec").Map()
		compPath := gjson.Get(config, "_components."+componentName+".path").String()

		// spec is missing // FIXME - this should be fatal, but we are skipping for now
		if len(spec) == 0 {
			log.Warn().Str("cluster", clusterName).
				Str("component", componentName).
				Msg("Component has no kr8_spec. Skipped!")
			continue
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
		} else  {
			vm.ExtCode("kr8", config+"."+componentName)
		}

		// add kr8_allparams extcode with all cluster params
		if spec["enable_kr8_allparams"].Bool() {
			// include full render of all component params
			if allconfig == "" {
				// only do this if we have not already cached it and don't already have it stored
				if components == "" {
					// all component params are in config
					allconfig = config
				} else {
					allconfig = renderClusterParams(cmd, clusterName, []string{}, clusterParams, false)
				}
			}
			vm.ExtCode("kr8_allparams", allconfig)
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
		// create or clean component dir
		if _, err := os.Stat(componentDir); os.IsNotExist(err) {
			err := os.Mkdir(componentDir, os.ModePerm)
			if err != nil {
				log.Fatal().Err(err).Msg("")
			}
		} else {
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
				if filepath.Ext(name) == ".yaml" {
					err = os.RemoveAll(filepath.Join(componentDir, name))
					if err != nil {
						log.Fatal().Err(err).Msg("")
					}
				}
			}
		}

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
				if inc_spec["dest_dir"].Exists()  {
					// handle alternate output directory for file
					altdir := inc_spec["dest_dir"].String()
					// dir is always relative to generate dir
					outputDir = clGenerateDir + "/" + altdir
					// ensure this directory exists
					if _, err := os.Stat(outputDir); os.IsNotExist(err) {
						err = os.Mkdir(outputDir, os.ModePerm)
						if err != nil {
							log.Fatal().Err(err).Msg("")
						}
					}
				}
				if inc_spec["dest_name"].Exists()  {
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
				log.Fatal().Err(err).Msg("Error evaluating jsonnet snippet")
			}

			// write output to generated files as yaml stream
			f, err := os.Create(outputFile)
			if err != nil {
				log.Fatal().Err(err).Msg("")
			}
			defer f.Close()

			var o []interface{}
			if err := json.Unmarshal([]byte(j), &o); err != nil {
				log.Fatal().Err(err).Msg("")
			}
			for _, jobj := range o {
				_, err := f.WriteString("---\n")
				if err != nil {
					log.Fatal().Err(err).Msg("")
				}
				buf, err := goyaml.Marshal(jobj)
				if err != nil {
					log.Fatal().Err(err).Msg("")
				}
				_, err = f.WriteString(string(buf) + "\n")
				if err != nil {
					log.Fatal().Err(err).Msg("")
				}
			}
		}
	}
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate components",
	Long:  `Generate components in clusters`,

	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {

		var clusterList []string
		if clusters == "" {
			// default is all clusters
			cd, err := getClusters(clusterDir)
			if err != nil {
				log.Fatal().Err(err).Msg("Error getting list of clusters")
			}
			for _, c := range cd.Cluster {
				// FIXME add filtering here
				clusterList = append(clusterList, c.Name)
			}
		} else {
			// use list of clusters that was passed in
			clusterList = strings.Split(clusters, ",")
		}

		var wg sync.WaitGroup
		parallel, err := cmd.Flags().GetInt("parallel")
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
		p, _ := ants.NewPool(parallel)

		for _, clusterName := range clusterList {
			wg.Add(1)
			cl := clusterName
			_ = p.Submit(func() {
				defer wg.Done()
				genProcessCluster(cmd, cl)
			})
		}
		wg.Wait()
	},
}

func init() {
	RootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVarP(&clusterParams, "clusterparams", "", "", "provide cluster params as single file - can be combined with --cluster to override cluster")
	generateCmd.Flags().StringVarP(&clusters, "clusters", "", "", "clusters to generate")
	generateCmd.Flags().StringVarP(&components, "components", "", "", "components to generate")
	generateCmd.Flags().StringVarP(&generateDir, "generate-dir", "", "", "output directory")
	generateCmd.Flags().IntP("parallel", "", 1, "parallelism")
}
