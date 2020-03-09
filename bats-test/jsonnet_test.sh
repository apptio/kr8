#!/usr/bin/env bats

# FIXME: Add --prune tests

if [ -z "$KR8" ]; then
  KR8=kr8
fi

KR8_ARGS="-d data"
CLUSTER=bats

@test "Check jsonnet json parsing" {
  expected=$(<expected/jsonnet_basic_json)
  run $KR8 $KR8_ARGS -c $CLUSTER jsonnet render data/misc/basic.json
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check jsonnet basic jsonnet parsing" {
  expected=$(<expected/jsonnet_basic_jsonnet)
  run $KR8 $KR8_ARGS -c $CLUSTER jsonnet render data/misc/basic.jsonnet
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check jsonnet component parsing (default: json)" {
  expected=$(<expected/jsonnet_comp1_json)
  run $KR8 $KR8_ARGS jsonnet render -c bats -C comp1 data/components/comp1/comp1.jsonnet
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

# this is a bug where we stacktrace if --component isn't set
# FIXME: could be better
@test "Check jsonnet parsing without component - FAIL" {
  expected=$(<expected/jsonnet_comp1_jsonnet)
  run $KR8 $KR8_ARGS jsonnet render -c bats data/components/comp1/comp1.json
  [ "$status" -eq 2 ]
}

# Explicit formats
@test "Check jsonnet component parsing (format: json)" {
  expected=$(<expected/jsonnet_comp1_json)
  run $KR8 $KR8_ARGS jsonnet render -c bats -C comp1 -F json data/components/comp1/comp1.jsonnet
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check jsonnet component parsing (format: yaml)" {
  expected=$(<expected/jsonnet_comp1_yaml)
  run $KR8 $KR8_ARGS jsonnet render -c bats -C comp1 -F yaml data/components/comp1/comp1.jsonnet
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

# stream format with one object is a stacktrace
# FIXME: could be better
@test "Check jsonnet component parsing (format: stream) - FAIL" {
  expected=$(<expected/jsonnet_comp1_json)
  run $KR8 $KR8_ARGS jsonnet render -c bats -C comp1 -F stream data/components/comp1/comp1.jsonnet
  [ "$status" -eq 2 ]
}

# List of objects
@test "Check jsonnet list component parsing (format: json)" {
  expected=$(<expected/jsonnet_comp1_list_json)
  run $KR8 $KR8_ARGS jsonnet render -c bats -C comp1 -F json data/components/comp1/comp1_list.jsonnet
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check jsonnet list component parsing (format: yaml)" {
  expected=$(<expected/jsonnet_comp1_list_yaml)
  run $KR8 $KR8_ARGS jsonnet render -c bats -C comp1 -F yaml data/components/comp1/comp1_list.jsonnet
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check jsonnet list component parsing (format: stream)" {
  expected=$(<expected/jsonnet_comp1_list_stream)
  run $KR8 $KR8_ARGS jsonnet render -c bats -C comp1 -F stream data/components/comp1/comp1_list.jsonnet
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

