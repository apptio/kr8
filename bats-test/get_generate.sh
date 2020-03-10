#!/bin/bash

if [ -z "$KR8" ]; then
  KR8=kr8
fi

KR8_ARGS="-d data"
CLUSTER=bats

# "get" are different, from "cluster", in a debug line that probably shouldn't exist
$KR8 $KR8_ARGS get params --cluster $CLUSTER > expected/get_params
$KR8 $KR8_ARGS get params --cluster $CLUSTER -C comp1 > expected/get_params_comp1
$KR8 $KR8_ARGS get params --cluster $CLUSTER -P comp2 > expected/get_params_comp2

