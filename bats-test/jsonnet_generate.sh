#!/bin/bash

if [ -z "$KR8" ]; then
  KR8=kr8
fi

CLUSTER=bats
KR8_ARGS="-d data -c $CLUSTER"

# Basic tests, no cluster imports
$KR8 $KR8_ARGS jsonnet render data/misc/basic.json > expected/jsonnet_basic_json
$KR8 $KR8_ARGS jsonnet render data/misc/basic.jsonnet > expected/jsonnet_basic_jsonnet

# Cluster/component parsing, json or yaml will handle single objects
$KR8 $KR8_ARGS jsonnet render -C comp1 -F json data/components/comp1/comp1.jsonnet > expected/jsonnet_comp1_json
$KR8 $KR8_ARGS jsonnet render -C comp1 -F yaml data/components/comp1/comp1.jsonnet > expected/jsonnet_comp1_yaml

# All formats handle list parsing
$KR8 $KR8_ARGS jsonnet render -C comp1 -F json data/components/comp1/comp1_list.jsonnet > expected/jsonnet_comp1_list_json
$KR8 $KR8_ARGS jsonnet render -C comp1 -F yaml data/components/comp1/comp1_list.jsonnet > expected/jsonnet_comp1_list_yaml
$KR8 $KR8_ARGS jsonnet render -C comp1 -F stream data/components/comp1/comp1_list.jsonnet > expected/jsonnet_comp1_list_stream

# With --clusterparams
$KR8 $KR8_ARGS jsonnet render -C comp2 -F yaml data/components/comp2/comp2.jsonnet \
  --clusterparams data/misc/cluster_params.jsonnet > expected/jsonnet_comp2_with_file_yaml
