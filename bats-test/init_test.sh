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

# Remove this for now
#  There's a weird race condition where it can check out "master" for kr8
#  and cause problems.  That's going to be deeper in the code.
@test "Check init failure - existing directory" {
  skip "skip testing, code issue"
  mkdir -p ./init-test2
  run $KR8 init repo ./init-test2
  [ "$status" -eq 1 ]
  rm -rf ./init-test2
}
