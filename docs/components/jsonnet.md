# Jsonnet Component

A very simple component might just be a few lines of jsonnet.

Consider the situation whereby you might have two clusters, one in AWS and one in DigitalOcean. You need to set a default storageclass. You could do this with jsonnet.

Your jsonnet component would look like this:

```bash
components/storageclasses
├── Taskfile.yml
├── params.jsonnet
├── storageclasses.jsonnet
```

## Taskfile

The taskfile for a component like would look like this:

```yaml
version: 2

tasks:
  fetch:
    desc: "fetch component dependencies"

  generate:
    desc: "generate"
    cmds:
      - kr8-helpers clean-output
      - kr8-helpers jsonnet-render storageclasses.jsonnet
```

Notice we still add a fetch task which is an empty command

## Params

As a reminder, every component requires a params file. We need to set a namespace for the component, even though it's a cluster level resource - namespace is a required paramater for kr8

```yaml
{
  namespace: 'kube-system',
  release_name: 'storageclasses',
}
```

## Jsonnet Manifest

Your jsonnet manifest looks like this:

```jsonnet
local config = std.extVar('kr8'); # imports the config from params.jsonnet
local kr8_cluster = std.extVar('kr8_cluster'); # a jsonnet external variable from kr8 that gets cluster values and data

# a jsonnet function for creating a storageclass
local StorageClass(name, type, default=false) = {
  apiVersion: 'storage.k8s.io/v1',
  kind: 'StorageClass',
  metadata: {
    name: name,
    annotations: {
      'storageclass.kubernetes.io/is-default-class': if default then 'true' else 'false',
    },
  },
  parameters: {
    type: type,
  },
};

# check the cluster configuration for a type, if it's AWS make a gp2 type storageclass
if kr8_cluster.cluster_type == 'aws' then kube.objectValues(
  {
    // default gp2 storage class, not tied to a zone
    ebs_gp2: StorageClass('gp2', 'gp2', true) {},
  }
) else [] # don't make a storageclass if it's not AWS
```


