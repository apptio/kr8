package cmd

import (
	//"fmt"
	//"fmt"
	log "github.com/Sirupsen/logrus"
	//"github.com/olekukonko/tablewriter"
	"os"
	"path/filepath"
	"strings"
)

func (c *Clusters) AddItem(item Cluster) Clusters {
	c.Cluster = append(c.Cluster, item)
	return *c
}

func getClusters(searchDir string) (Clusters, error) {

	fileList := make([]string, 0)
	//clusterName := make([]string, 0)
	//pathName := make([]string, 0)

	e := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		fileList = append(fileList, path)
		return err
	})

	if e != nil {
		log.Fatal("Error building cluster list: ", e)
	}

	ClusterData := []Cluster{}
	c := Clusters{ClusterData}

	for _, file := range fileList {

		splitFile := strings.Split(file, "/")
		// get the filename
		fileName := splitFile[len(splitFile)-1]

		if fileName == "cluster.jsonnet" {
			entry := Cluster{Name: splitFile[len(splitFile)-2], Path: strings.Join(splitFile[:len(splitFile)-1], "/")}
			c.AddItem(entry)

		}
	}

	return c, nil

}

func getCluster(searchDir string, clusterName string) string {

	fileList := make([]string, 0)
	//clusterName := make([]string, 0)
	//pathName := make([]string, 0)

	e := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		if strings.Contains(path, clusterName) {
			fileList = append(fileList, path)
			return nil
		} else {
			return err
		}
	})

	if e != nil {
		log.Fatal("Error building cluster list: ", e)
	}

	// Die if there's no cluster with name
	if len(fileList) < 1 {
		log.Fatal("Could not find cluster: ", clusterName)
	}

	// var for storing clusterpath
	var clusterPath string

	// range over the files
	for _, file := range fileList {

		// split the file string
		splitFile := strings.Split(file, "/")
		// get the filename
		fileName := splitFile[len(splitFile)-1]

		// if the filename == cluster.jsonnet..
		if fileName == "cluster.jsonnet" {
			clusterPath = file
		}
	}

	// return it
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

		// stop if we're in the basePath
		if rel == "." {
			break
		}

		// check if there's a params.json in the folder
		if _, err := os.Stat(targetDir + "/params.jsonnet"); err == nil {
			results = append(results, targetDir+"/params.jsonnet")
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
