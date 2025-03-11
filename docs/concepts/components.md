# Components

A component is a deployable unit that you wish to install to multiple clusters.

Your component might begin life before kr8 in one of a few ways:

  - a [Helm Chart](https://github.com/helm/charts/tree/master/stable)
  - a static [YAML manifest](https://github.com/kubernetes/examples/blob/master/guestbook/all-in-one/guestbook-all-in-one.yaml)
  - some [Jsonnet](https://github.com/coreos/prometheus-operator/tree/master/jsonnet/prometheus-operator)

but they all have something in common - you need to deploy it to multiple clusters with slight differences in configuration.


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

kr8's most useful feature is the ability to configure _parameters_ for a specific cluster.
It does that by specifying a `params.jsonnet` in each component.

Kr8 extracts core configuration parameters from the `kr8_spec` key.

Fields:

  - `namespace`: String. Required. The namespace the component should be installed in
  - `release_name`: String. Required. Analogous to a helm release - what the component should be called when it's installed into a cluster
  - `enable_kr8_allparams`: Bool. Optional. Includes a full render of all component params.  Used for components that reflect properties of other components.
  - `enable_kr8_allclusters`: Bool. Optional. Includes a full render of all cluster params.  Used for components that reflect properties of other clusters.
  - `jpaths`: List[filename]. Optional. Add additional jsonnet lib paths.
  - `disable_output_clean`: Bool. Optional. Purge any yaml files in the output dir that were not generated
  - `extfiles`: List[obj]. Optional.  Add additional files to load as jsonnet `ExtVar`s.  Example: `extfiles: [identifier: "filename.jsonnet", otherfile: "filename2.jsonnet" ]`
  - `includes`: List[string or obj]. Optional. Include and process additional files.  The `file` value must be a `jsonnet`, `yaml`, or `tpl` filename.
  Example:
  ```jsonnet
  includes: [
    "filename.jsonnet",
    {
      file: "filename.jsonnet",
      dest_dir: "altDir",
      dest_name: "altname1",
      dest_ext: "txt"
    },
    {
      file: "filename.jsonnet",
      dest_name: "altname2",
    },
    {
      file: "filename.jsonnet",
      dest_name: "altname3",
      dest_ext: "txt"
    },
  ]
  ```
  Will generate the following files:
  ```
  generate_dir:
    filename.jsonnet
    altname2.jsonnet
    altname3.txt
    altDir:
      altname1.txt
  ``` 

---

Without these parameters, components will not install a function. A barebones `params.jsonnet` would look like this:

```jsonnet
{
  kr8_spec: {
    namespace: 'kubemonkey',
    release_name: 'kubemonkey',
  },
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

