#!/bin/sh
set -eu
stack_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
CLOUDFLARE_API_TOKEN=$(security find-generic-password -w -s cloudflare-dns-tf-token)
export CLOUDFLARE_API_TOKEN
exec terraform -chdir="$stack_dir" "$@"
