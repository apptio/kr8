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

## The params tests also effectively test param hierarchy at all levels with "comp2"
## FIXME: Params above the cluster/<x>/params.jsonnet hierarchy bleed into the
##        params no matter which component is requested.  Is this correct?
##        "comp1" displays this behavior in a test in case it changes

@test "Check cluster params for all components" {
  expected=$(<expected/cluster_params)
  run $KR8 $KR8_ARGS cluster params -c "$CLUSTER"
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check cluster params for one component with cluster config (-C)" {
  expected=$(<expected/cluster_params_comp1)
  run $KR8 $KR8_ARGS cluster params -c "$CLUSTER" -C comp1
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check cluster params for one component only (-P)" {
  expected=$(<expected/cluster_params_comp2)
  run $KR8 $KR8_ARGS cluster params -c "$CLUSTER" -P comp2
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check cluster params with file override" {
  expected=$(<expected/cluster_params_file)
  run $KR8 $KR8_ARGS cluster params -c "$CLUSTER" --clusterparams data/misc/cluster_params.jsonnet
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

# Check behavior on a component that doesn't exist
@test "Check cluster params with unset component (-P)" {
  # This is wonky because of "echo" and fmt.Println, but matches anyway
  expected=""
  run $KR8 $KR8_ARGS cluster params -c "$CLUSTER" -P no_component
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

# Check behavior on a component that doesn't exist
@test "Check cluster params with unset component (-C)" {
  expected=$(<expected/cluster_params_no_comp)
  run $KR8 $KR8_ARGS cluster params -c "$CLUSTER" -C no_component
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

# --notunset has interesting behavior, and only exists on "cluster", not on "get"
@test "Check cluster params with unset component (-P) and --nounset flag - FAIL" {
  run $KR8 $KR8_ARGS cluster params -c "$CLUSTER" -P no_component --notunset
  [ "$status" -eq 1 ]
}

# But this *works* and gives cluster config + component list
@test "Check cluster params with unset component (-C) and --nounset flag" {
  expected=$(<expected/cluster_params_no_comp)
  run $KR8 $KR8_ARGS cluster params -c "$CLUSTER" -C no_component --notunset
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}
