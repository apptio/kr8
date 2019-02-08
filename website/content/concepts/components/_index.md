+++
title = "Components"
weight = 10
+++

# Components

A component is a deployable unit that you wish to install to multiple clusters.

Your component might begin life before kr8 in one of a few ways:

  - a [Helm Chart](https://github.com/helm/charts/tree/master/stable)
  - a static [YAML manifest](https://github.com/kubernetes/examples/blob/master/guestbook/all-in-one/guestbook-all-in-one.yaml)
  - some [Jsonnet](https://github.com/coreos/prometheus-operator/tree/master/jsonnet/prometheus-operator)

but they all have something in common - you need to deploy it to multiple clusters with slight differences in configuration.

## Component components

There are several key pieces that make up a component before you write a line of configuration.

## Taskfile

This task file lives inside the component directory. It should contain two tasks:

  - fetch - a manually run task which downloads all the components' dependencies (for example, helm chart or static manifest)
  - generate - this is the task that's run when kr8 generates the manifest for the cluster


These tasks will be highly dependent on the particular component - for example, a component using a helm chart will generally have a different set of fetch and generate tasks to a component using a static manifest.

An example Taskfile might look like this:

```yaml
version: 2

vars:
  KR8_COMPONENT: kubemonkey

tasks:
  fetch:
    desc: "fetch component kubemonkey"
    cmds:
      - curl -L https://github.com/asobti/kube-monkey/tarball/master > kubemonkey.tar.gz # download the local helm chart from the git repo
      - tar --strip-components=2 -xzvf kubemonkey.tar.gz asobti-kube-monkey-{{.GIT_COMMIT}}/helm # extract it
      - mv kubemonkey charts # place it in a charts directory
      - rm -fr *.tar.gz # remove the tar.gz from the repo
    vars:
      GIT_COMMIT:
        sh: curl -s https://api.github.com/repos/asobti/kube-monkey/commits/master | jq .sha -r | xargs git rev-parse --short

  generate:
    desc: "generate"
    cmds:
      - KR8_COMPONENT={{.KR8_COMPONENT}} kr8-helpers clean-output # clean the tmp directories each time we generate
      - KR8_COMPONENT={{.KR8_COMPONENT}} kr8-helpers helm-render-with-patch "{{.KR8_COMPONENT}}" patches.jsonnet # our generate command, which in this case is a helm-render with some patches in a jsonnet file
```

## Params

kr8's most useful feature is the ability to configure _parameters_ for a specific cluster. It does that by specifying a `params.jsonnet` in each component.

There are some *required* parameters which always must exist. They are:

  - `namespace`: the namespace the component should be installed in
  - `release_name`: analogous to a helm release - what the component should be called when it's installed into a cluster
  - `kubecfg_gc_enable`: whether this component should be garbage collected when the deployer script cleans up this component (generally should be false for important system components like namespaces)


Without these parameters, components will not install a function. A barebones `params.jsonnet` would look like this:

```jsonnet
{
  namespace: 'kubemonkey',
  release_name: 'kubemonkey',
  kubecfg_gc_enable: true,
}
```

### Cluster specific parameters

Once you start to install components into clusters, you'll want to specify parameters of your own.

These are done in the `params.jsonnet` and you can either specify a default, or make it mandatory using jsonnet's `error`.

Here's an more detailed example:

```jsonnet
{
  namespace: 'kubemonkey',
  release_name: 'kubemonkey',
  kubecfg_gc_enable: true,
  dry_run: false,
  run_hour: error 'Must specify a time for kubemonkey to run'
}
```

## Manifests

The final part of your component is the actual manifest generation. This depends on the source you're generating from. You can find more information about how to generate manifests in the [advanced components]( {{< relref "advanced/components" >}}) section of the documentation
