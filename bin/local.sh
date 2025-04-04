#!/bin/bash

#
# disco.sh
# installs the disco command line tool with go install and copies disco.yml to
# the user's home config directory
#

die() { printf "that ain't it, %s" "$*"; exit 7; }

[[ $(head -n1 go.mod) == "module github.com/dedelala/disco" ]] || die "go mod"
[[ -f disco.yml ]] || die "no disco.yml found at repository root"

go run ./cmd/discod/main.go -c disco.yml -l ":8080" -v debug
