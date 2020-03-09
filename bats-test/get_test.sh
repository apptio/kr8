#!/usr/bin/env bats

if [ -z "$KR8" ]; then
  KR8=kr8
fi

KR8_ARGS="-d data"
CLUSTER=bats

# NOTE: These are expected to be the same as "cluster ..." output, so reuse
# the expected files.  --clusterparams might throw a wrench in this

@test "Check get clusters output" {
  expected=$(<expected/cluster_list)
  run $KR8 $KR8_ARGS get clusters
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check get components output" {
  expected=$(<expected/cluster_components)
  run $KR8 $KR8_ARGS get components --cluster "$CLUSTER"
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

# These have a (debug?) output line in the stock version
@test "Check get params for all components" {
  expected=$(<expected/get_params)
  run $KR8 $KR8_ARGS get params -c "$CLUSTER"
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check get params for one component" {
  expected=$(<expected/get_params_comp1)
  run $KR8 $KR8_ARGS get params -c "$CLUSTER" -C comp1
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}
