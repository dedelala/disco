#!/bin/bash

die() { printf "that ain't it, %s" "$*"; exit 7; }

if ! [[ $(head -n1 go.mod) == "module github.com/dedelala/disco" ]]; then
    die "go mod"
fi

[[ -f disco.yml ]] || die "no disco.yml found at repository root"

CGO_ENABLED=0 go install ./cmd/disco/. || die "build"

mkdir -p "$HOME/.config" || die "config dir"
cp disco.yml "$HOME/.config/" || die "disco.yml"

