#!/bin/bash

#
# lifx-products.sh
# fetches the latest lifx products data from github and writes to
# lifx/products.json
#

die() { printf "twasn't meant to be, %s" "$*"; exit 7; }

[[ $(head -n1 go.mod) == "module github.com/dedelala/disco" ]] || die "go mod"

curl -f -o lifx/products.json https://raw.githubusercontent.com/LIFX/products/refs/heads/master/products.json || die "fetch products.json"
