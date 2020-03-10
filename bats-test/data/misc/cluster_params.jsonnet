{
  _cluster+: {
    cluster_name: 'bats2',
    cluster_params_key: 'here',
  },
  _components+: {
    comp3: { path: 'components/comp1' },
  },
  comp2+: {
    p_component: 'args',
    p_cluster: 'args',
    p_cluster_jsonnet: 'args',
    p_site_params: 'args',
  },
}
