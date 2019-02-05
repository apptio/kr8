+++
title = "About"
weight = 10
+++

# Overview

kr8 is a configuration management tool for Kubernetes, designed to generate deployable manifests for the components required to make your clusters _usable_.

Its main function is to translate and manipulate YAML or JSON without using a templating engine.

Its flexibility means you can perform operations such as:

  - Use existing helm charts, and patch the resources in them
  - Use other overlay tools, like [Kustomize](https://github.com/kubernetes-sigs/kustomize)
  - Create resources in pure JSON or Jsonnet

## Goals & Focus

The focus with kr8 is _not_ to make deploying applications easier. We believe this problem is already being tackled quite widely and with excellent success rates.

The main aim of kr8 is to allow Cluster operators to generate the manifests needed to deploy key cluster components to different clusters.

It's designed to prevent configuration mismatches between (for example) development and production clusters and to prevent snowflake cluster deployments.





