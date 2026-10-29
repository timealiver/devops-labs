#!/bin/bash
b=/bundles/$1; shift
tmp=$(mktemp)
jq --argjson a "$(printf "%s\n" "$@" | jq -R . | jq -s .)" \
  ".process.args = \$a | .process.terminal = false | .process.user = {uid:(env.GVUID//\"0\"|tonumber),gid:(env.GVUID//\"0\"|tonumber)} | .root.readonly = false" \
  $b/config.json > $tmp && cat $tmp > $b/config.json
exec runsc --ignore-cgroups --network=${GVNET:-none} --rootless=false run -bundle $b gv$$
