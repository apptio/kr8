# kr8

kr8 is a very opinionated tool used for rendering [jsonnet](http://jsonnet.org) manifests for numerous Kubernetes clusters.

It has been designed to work like a simple configuration management framework, allowing operators to specify configuration for _components_ across multiple clusters.

kr8 is a work in progress (currently in Alpha), but is in use at Apptio for managing components of multiple Kubernetes clusters.

For more information about the inspiration and the problem kr8 solves, check out this [blog post](https://leebriggs.co.uk/blog/2018/05/08/kubernetes-config-mgmt.html).

kr8 consists of:

 - kr8 - a Go binary for rendering manifests
 - [Task](https://github.com/go-task/task) - a third party Go binary for executing tasks
 - Configs - A configuration directory which contains config for clusters and the components installed into those clusters

kr8 is not designed to be a tool to help you install and deploy applications. It's specifically designed to manage and maintain configuration for the cluster level services. For more information, see the [components](##components) section.

In order to use kr8, you'll need a configuration repository to go with this binary. See the [example](https://github.com/apptio/kr8-configs) repo for more information.

# Features

 - Generate and customize component configuration for Kubernetes clusters across environments, regions and platforms
 - Opinionated config, flexible deployment. kr8 simply generates manifests for you, you decide how to deploy them
 - Render and override component config from multiple sources, such as Helm, Kustomize and static manifests
 - CI/CD friendly

# Concepts & Tools

## Component

A component is something you install in your cluster to make it function and work as you expect. Some examples of components might be:

 - [cert-manager](https://github.com/jetstack/cert-manager)
 - [nginx-ingress](https://github.com/kubernetes/ingress-nginx)
 - [sealed-secrets](https://github.com/bitnami-labs/sealed-secrets)

Components are _not_ the applications you want to run in your cluster. Components are generally applications you'd run in your cluster to make those applications function and work as expected.

## Clusters

A cluster is a Kubernetes cluster running in a cloud provider, datacenter or elsewhere. You will more than likely have multiple clusters across multiple environments and regions.

Clusters have:

  - cluster configuration, which can be used as part of the Jsonnet configuration later. This consists of things like the cluster name, type, region etc
  - components, which you'd like to install in a cluster
  - component configuration, which is modifications to a component which are specific to a cluster. An example of this might be the path to an SSL certificate for the nginx-ingress controller, which may be different across cloud providers.

## Taskfiles

Instead of reinventing the wheel, the kr8 ecosystem makes uses of the [Task](https://github.com/go-task/task) tool to generate the cluster configuration. The task files generally call kr8 in order to render the manifests for a component. We chose task because it supports yaml and json configuration files, which mean we can continue to leverage jsonnet to write taskfiles where needed.

## Jsonnet

All configuration for kr8 is written in [Jsonnet](https://jsonnet.org/). Jsonnet was chosen because it allows us to use code for configuration, while staying as close to JSON as possible.

# Building

See the [docs](docs/BUILDING.md)

# Contributing

Fork the repo in github and send a merge request!

# Caveats

There are currently no tests, and the code is not very [DRY](https://en.wikipedia.org/wiki/Don%27t_repeat_yourself).

This was (one of) Apptio's first exercise in Go, and pull requests are very welcome.
  



