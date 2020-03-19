{
  _cluster+: {
    cluster_name: 'bats',
    cluster_type: 'fake',
  },
  _components+: {
    comp1: { path: 'components/comp1' },
    comp2: { path: 'components/comp2' },
  },
  // Don't do this, but it works.
  comp2+: {
    p_cluster_jsonnet: 'cluster_jsonnet',
  },
}
