#!/usr/bin/env bats

# FIXME: Add --prune tests

if [ -z "$KR8" ]; then
  KR8=kr8
fi

KR8_ARGS="-d data"
CLUSTER=bats

@test "Check render jsonnet json parsing" {
  expected=$(<expected/jsonnet_basic_json)
  run $KR8 $KR8_ARGS -c $CLUSTER render jsonnet data/misc/basic.json
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check render jsonnet basic jsonnet parsing" {
  expected=$(<expected/jsonnet_basic_jsonnet)
  run $KR8 $KR8_ARGS -c $CLUSTER render jsonnet data/misc/basic.jsonnet
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check render jsonnet component parsing (default: json)" {
  expected=$(<expected/jsonnet_comp1_json)
  run $KR8 $KR8_ARGS render jsonnet -c bats -C comp1 data/components/comp1/comp1.jsonnet
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

# this is a bug where we stacktrace if --component isn't set
# FIXME: could be better
@test "Check render jsonnet parsing without component - FAIL" {
  expected=$(<expected/jsonnet_comp1_json)
  run $KR8 $KR8_ARGS render jsonnet -c bats data/components/comp1/comp1.jsonnet
  [ "$status" -eq 2 ]
}

# Explicit formats
@test "Check render jsonnet component parsing (format: json)" {
  expected=$(<expected/jsonnet_comp1_json)
  run $KR8 $KR8_ARGS render jsonnet -c bats -C comp1 -F json data/components/comp1/comp1.jsonnet
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check render jsonnet component parsing (format: yaml)" {
  expected=$(<expected/jsonnet_comp1_yaml)
  run $KR8 $KR8_ARGS render jsonnet -c bats -C comp1 -F yaml data/components/comp1/comp1.jsonnet
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

# stream format with one object is a stacktrace
# FIXME: could be better
@test "Check render jsonnet component parsing (format: stream) - FAIL" {
  expected=$(<expected/jsonnet_comp1_json)
  run $KR8 $KR8_ARGS render jsonnet -c bats -C comp1 -F stream data/components/comp1/comp1.jsonnet
  [ "$status" -eq 2 ]
}

# List of objects
@test "Check render jsonnet list component parsing (format: json)" {
  expected=$(<expected/jsonnet_comp1_list_json)
  run $KR8 $KR8_ARGS render jsonnet -c bats -C comp1 -F json data/components/comp1/comp1_list.jsonnet
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check render jsonnet list component parsing (format: yaml)" {
  expected=$(<expected/jsonnet_comp1_list_yaml)
  run $KR8 $KR8_ARGS render jsonnet -c bats -C comp1 -F yaml data/components/comp1/comp1_list.jsonnet
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check render jsonnet list component parsing (format: stream)" {
  expected=$(<expected/jsonnet_comp1_list_stream)
  run $KR8 $KR8_ARGS render jsonnet -c bats -C comp1 -F stream data/components/comp1/comp1_list.jsonnet
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

# Test with --clusterparams
@test "Check render jsonnet parsing with --clusterparams" {
  expected=$(<expected/jsonnet_comp2_with_file_stream)
  run $KR8 $KR8_ARGS render jsonnet -C comp2 -F yaml data/components/comp2/comp2.jsonnet \
    --clusterparams data/misc/cluster_params.jsonnet
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

# FIXME: stacktrace if we call a component that doesn't exist in the --clusterparams file
#        even if that component exists and has its own params
#        Only the clusterprams file gets used, even blanking other params
@test "Check render jsonnet parsing with --clusterparams" {
  #expected=$(<expected/jsonnet_comp2_with_file_stream)
  run $KR8 $KR8_ARGS render jsonnet -C comp1 -F stream data/components/comp2/comp1_list.jsonnet \
    --clusterparams data/misc/cluster_params.jsonnet
  [ "$status" -eq 2 ]
  #diff <(echo "$output") <(echo "$expected")
}

# helmclean tests - use "yaml" command generated files

# Stacktrace on bad YAML
# FIXME: could be better
@test "Check render helmclean on bad YAML  - FAIL" {
  run $KR8 $KR8_ARGS render helmclean < data/misc/fail.yaml
  [ "$status" -eq 2 ]
}

# Stacktrace if we don't match "kind" or other k8sy things
# FIXME: could be better
@test "Check render helmclean object without kind - FAIL" {
  run $KR8 $KR8_ARGS render helmclean < data/misc/nokind.yaml
  [ "$status" -eq 2 ]
}
@test "Check render helmclean stream with no nulls" {
  expected=$(<expected/yaml_helmclean_clean)
  run $KR8 $KR8_ARGS render helmclean < data/misc/clean.yaml
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check render helmclean stream with nulls" {
  # we are explicitly expecting the "clean" output to match
  expected=$(<expected/yaml_helmclean_clean)
  run $KR8 $KR8_ARGS render helmclean < data/misc/dirty.yaml
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}
