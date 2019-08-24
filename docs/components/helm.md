# Helm

kr8 can render helm charts locally and inject parameters as helm values. This provides a great degree of flexibility when installing components into clusters.

## Taskfile

An example taskfile for a helm chart might look like this:

```yaml
version: 2

vars:
  CHART_VER:  v0.8.1
  CHART_NAME: cert-manager

tasks:
  fetch:
    desc: "fetch component dependencies"
    cmds:
      - rm -fr charts vendored; mkdir charts vendored
      # add the helm repo and fetch it locally into the charts directly
      - helm fetch --repo https://charts.jetstack.io --untar --untardir ./charts --version "{{.CHART_VER}}" "{{.CHART_NAME}}"
      - wget --quiet -N https://raw.githubusercontent.com/jetstack/cert-manager/release-0.8/deploy/manifests/00-crds.yaml -O - | grep -v ^# > vendored/00cert-manager-crd.yaml


  generate:
    desc: "generate"
    cmds:
      - kr8-helpers clean-output
      - kr8-helpers yaml-install vendored/00cert-manager-crd.yaml
      - kr8-helpers jsonnet-render 00namespace.jsonnet
      - kr8-helpers helm-render "{{.CHART_NAME}}"
```

## Params

The `params.jsonnet` for a helm chart directory should include the helm values you want to use. Here's an example:

```jsonnet
{
  release_name: 'cert-manager',
  namespace: 'cert-manager',
  kubecfg_gc_enable: true,
  kubecfg_update_args: '--validate=false',
  helm_values: {
    webhook: { enabled: false },  // this is a value for the helm chart
  },
}
```

## Values file

A values file is a required file for a helm component. The name of the file must be `componentname-values.jsonnet` (for example: cert-manager-values.jsonnet). It's content would be something like this:

```jsonnet
local config = std.extVar('kr8');

if 'helm_values' in config then config.helm_values else {}
```

You can also optionally set the values for the helm chart in here, this would look something like this:

```jsonnet
{
  replicaCount: 2
}
```

## Patches

There are certain situations where a configuration option is not available for a helm chart, for example, you might want to add an argument that hasn't quite made it into the helm chart yet, or add something like [pod affinity](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity) where it isn't actually a value option in a helm chart.

kr8 helps you in this situation by providing a mechanism to patch a helm chart. You need to use the `helm-render-with-patch` helper and provide a `patches.jsonnet` in the component directory.

Here's an example `patches.jsonnet` for [external-dns](https://github.com/kubernetes-incubator/external-dns)

```jsonnet
local apptio = import 'apptio.libsonnet';
local helpers = import 'helpers.libsonnet';  // some helper functions
local kube = import 'kube.libsonnet';
local config = std.extVar('kr8'); // config is all the config from params.jsonnet

// remove Secret objects and add a namespace
[
  for object in helpers.list(
    // object list is converted to hash of named objects, then they can be modified by name
    helpers.named(helpers.helmInput) + {
      ['Deployment/' + config.release_name]+: helpers.patchContainer({
        // Injects an extra arg, which wasn't originally in the helm chart
        [if std.objectHas(config,'assumeRoleArn') then 'args']+: ['--aws-assume-role='+config.assumeRoleArn],
      }),
    },
  )
]
```
