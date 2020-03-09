#!/usr/bin/env bats

if [ -z "$KR8" ]; then
  KR8=kr8
fi

KR8_ARGS="-d data"
CLUSTER=bats

# Stacktrace on bad YAML
# FIXME: could be better
@test "Check yaml helmclean on bad YAML  - FAIL" {
  run $KR8 $KR8_ARGS yaml helmclean < data/misc/fail.yaml
  [ "$status" -eq 2 ]
}

# Stacktrace if we don't match "kind" or other k8sy things
# FIXME: could be better
@test "Check yaml helmclean object without kind - FAIL" {
  run $KR8 $KR8_ARGS yaml helmclean < data/misc/nokind.yaml
  [ "$status" -eq 2 ]
}
@test "Check yaml helmclean stream with no nulls" {
  expected=$(<expected/yaml_helmclean_clean)
  run $KR8 $KR8_ARGS yaml helmclean < data/misc/clean.yaml
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}

@test "Check yaml helmclean stream with nulls" {
  # we are explicitly expecting the "clean" output to match
  expected=$(<expected/yaml_helmclean_clean)
  run $KR8 $KR8_ARGS yaml helmclean < data/misc/dirty.yaml
  [ "$status" -eq 0 ]
  diff <(echo "$output") <(echo "$expected")
}
