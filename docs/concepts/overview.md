# Concepts

kr8 has two main concepts you should be aware of before you get started:

  - [components](components.md)
  - [clusters](clusters.md)

The relationship between components and clusters are simple: components are installed on clusters. You will have components that are installed on all clusters, and some components will only be installed on _some_ clusters.

Components can be declared multiple times within a cluster, as long as they are named distinctly.

Clusters are unique and singular. They have a name which is specified via the directory structure under `clusters`

Read more about [components](components.md) and [clusters](clusters.md)
