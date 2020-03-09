#!/bin/bash

if [ -z "$KR8" ]; then
  KR8=kr8
fi

# helmclean reads from stdin
$KR8 yaml helmclean < data/misc/clean.yaml > expected/yaml_helmclean_clean
