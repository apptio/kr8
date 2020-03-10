#!/usr/bin/env bats

if [ -z "$KR8" ]; then
  KR8=kr8
fi

@test "Check init success" {
  rm -rf ./init-test
  run $KR8 init repo ./init-test
  [ "$status" -eq 0 ]
  [ -d "init-test/clusters" ]
  [ -d "init-test/components" ]
  [ -d "init-test/lib" ]
  rm -rf ./init-test
}

@test "Check init failure - existing directory" {
  mkdir -p ./init-test
  run $KR8 init repo ./init-test
  [ "$status" -eq 1 ]
  rm -rf ./init-test
}
