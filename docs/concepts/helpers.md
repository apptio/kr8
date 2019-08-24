# Helpers

kr8-helpers is a set of scripts that wraps around the kr8 command for rendering components. The different types of components have different helps.

You should ensure the `kr8-helpers` scripts are available in your `$PATH`. You can find them [here](https://github.com/apptio/kr8/tree/master/scripts)

## clean-output

This helper cleans the generated directory and the metadata directory for each component. It should generally be the first helper you run as part of your component Taskfile

## helm-render

Render a helm chart using `helm template`. This allows you to install helm charts as components with kr8. For more information, see the [helm component section](../components/helm.md)

Example:

```bash
kr8-helpers helm-render "{{.CHART_NAME}}"
```

## helm-render-with-patch

Similar to `helm-render` except you can also patch the helm chart and add options and configuration that might not be exposed as a helm `values.yaml`. This helper will look for a `patches.jsonnet` inside the component directory. For more information, see the [helm component section](../components/helm.md)

Example:

```bash
kr8-helpers helm-render-with-patch "{{.CHART_NAME}}" patches.jsonnet
```

## yaml-install

The simplest helper, this just copies a specified yaml file for the component into the generated directory. It also cleans the yaml file of any unnecessary whitespace using the `helmclean` command in kr8

Example:

```bash
kr8-helpers yaml-install vendored/01_crd.yaml
```

## jsonnet-render

Render a jsonnet file. If you're starting a component without any source manifests or helm chart, or using something like the [prometheus-operator](https://github.com/coreos/prometheus-operator/tree/master/jsonnet/prometheus-operator) this would be what you'd use.

Example:

```bash
kr8-helpers jsonnet-render secrets.jsonnet
```

## jk-render

[jk](https://jkcfg.github.io/#/) is a tool which allows you to write configuration as actual code. You can use `jk` to render a component:

Example:

```bash
kr8-helpers jk-render nginx.js
```





