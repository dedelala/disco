#!/bin/bash

die() { printf "that ain't it, %s" "$*"; exit 7; }

if ! [[ $(head -n1 go.mod) == "module github.com/dedelala/disco" ]]; then
    die "go mod"
fi

[[ -n $REMOTE ]] || die "please set REMOTE ssh host"
[[ -f disco.yml ]] || die "no disco.yml found at repository root"

rm -rf dist
mkdir -p dist

CGO_ENABLED=0 GOARCH=arm64 go build -o dist/discod ./cmd/discod/. || die "build"

scp dist/discod "$REMOTE:" || die "bin"
scp disco.yml "$REMOTE:" || die "yml"

ssh "$REMOTE" sudo sv down discod || die "sv down"
ssh "$REMOTE" sudo cp discod /usr/local/bin/ || die "bin 2"
ssh "$REMOTE" sudo setcap CAP_NET_BIND_SERVICE=+eip /usr/local/bin/discod || die "setcap"
ssh "$REMOTE" sudo cp disco.yml /etc/ || die "yml 2"
ssh "$REMOTE" sudo sv up discod || die "sv up"

