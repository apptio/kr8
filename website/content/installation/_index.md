+++
title = "Installation"
weight = 20
+++

# Installation

## Github Releases


We build binaries for OS X and Linux using [goreleaser](https://goreleaser.com/)

Simply head over to the [releases page](https://github.com/apptio/kr8/releases) to grab the latest version.

## Homebrew

We maintain a [homebrew tap](https://github.com/apptio/homebrew-tap) which makes installing on OS X a breeze. Simply add the tap and install:

```bash
brew tap apptio/tap
brew install kr8
```

# Dependencies

The kr8 binary is currently just one of a set of required dependencies. We plan to explore the need for these dependencies as time goes on, but right now, yhey are required.

## go-task

go-task is used inside components and for cluster config generation. It was chosen over Make because it supports parallel builds and JSON files as input (meaning we can use jsonnet to build the files, if needed).

### Installation

Check out the [go-task install docs](https://github.com/go-task/task/blob/master/docs/installation.md) for information on installing go-task

## Helm

If you plan on rendering and manipulating helm charts in your kr8-configs, you'll need the helm command line tool installed.

### Installation

Check out the [helm installation docs](https://github.com/helm/helm/blob/master/docs/install.md) for information on installing go-task

{{% alert theme="warning" %}}
Helm will not work unless you initialize it on your client machine.

By default, it'll ask you to run `helm init` - which will install the tiller component and presents a security risk

You should use `helm init --client-only` for kr8
{{% /alert %}}
