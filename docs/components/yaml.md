# YAML Component

kr8 can use a static k8s manifest as a source input. You can then manipulate the structure of that YAML using Jsonnet. kr8 takes care of the heavy lifting for you.

## Taskfile

You'll need a taskfile that downloads the original manifests for you in the `fetch` task. Here's an example:

```yaml

version: 2

# Download the example manifests for the metrics server.
# This creates a directory, "vendored" which contains the manifests
tasks:
  fetch:
    desc: "fetch component dependencies"
    cmds:
      - rm -rf vendored
      - mkdir -p vendored sources
      - git clone --depth=1 https://github.com/kubernetes-incubator/metrics-server sources/metrics-server
      - cp -a sources/metrics-server/deploy/1.8+/*.yaml ./vendored/
      - git add ./vendored
      - rm -rf sources

  generate:
    desc: "generate"
    cmds:
      - kr8-helpers clean-output
      - find vendored -type f -name "*.yaml" -exec kr8-helpers yaml-install '{}' \; # install all the manifests in the vendored directory directly, without changing then
      # use the metrics-server-deployment.yaml as the input to the jsonnet file
      - KR8_JSONNET_ARGS='--ext-str-file inputMetricsServerDeploy=vendored/metrics-server-deployment.yaml' kr8-helpers jsonnet-render metrics-server-deployment.jsonnet
```

## Jsonnet

You'll notice in the taskfile above that this line:

```
KR8_JSONNET_ARGS='--ext-str-file inputMetricsServerDeploy=vendored/metrics-server-deployment.yaml' kr8-helpers jsonnet-render metrics-server-deployment.jsonnet
```

References one of the files in vendored. This give us the ability to modify this file. Here's how the jsonnet looks:

```jsonnet
local helpers = import 'helpers.libsonnet';
local parseYaml = std.native('parseYaml');
# this must match the `ext-str-file` value in the taskfile
# it imports those values with the variable name "deployment"
local deployment = parseYaml(std.extVar('inputMetricsServerDeploy'));

local args = [
  "--kubelet-insecure-tls",
  "--kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname",
]

[
  # drop all the secrets if they're found, we don't want to check them into git
  if object.kind == 'Secret' then {} else object
  for object in helpers.list(
    helpers.named(deployment) + {
      # grab kind deployment with name metrics-server, and add some more args
      ['Deployment/kube-system/metrics-server']+: helpers.patchContainerNamed(
        "metrics-server",
        {
          "args"+: args,
        }
      ),
    }
  )
]

```
