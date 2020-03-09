#!/usr/bin/env bats

if [ -z "$KR8" ]; then
  KR8=kr8
fi

KR8_ARGS="-d data"
CLUSTER=bats

@test "Check cluster list output" {
  expected=$(<expected/cluster_list)
  run $KR8 $KR8_ARGS cluster list
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check cluster components output" {
  expected=$(<expected/cluster_components)
  run $KR8 $KR8_ARGS cluster components --cluster "$CLUSTER"
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check cluster params for all components" {
  expected=$(<expected/cluster_params)
  run $KR8 $KR8_ARGS cluster params -c "$CLUSTER"
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check cluster params for one component" {
  expected=$(<expected/cluster_params_comp1)
  run $KR8 $KR8_ARGS cluster params -c "$CLUSTER" -C comp1
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}
