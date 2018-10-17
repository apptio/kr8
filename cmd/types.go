package cmd

// init a struct for a single item
type Cluster struct {
	Name string
	Path string
}

// init a grouping struct
type Clusters struct {
	Cluster []Cluster
}
