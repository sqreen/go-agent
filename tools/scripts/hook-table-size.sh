#!/usr/bin/env bash
scripts_dir=$(dirname $BASH_SOURCE)

size=0
while read -r segment; do
    cols=( $segment )
    size=$((size+${cols[1]}))
done < <(go tool nm -size $1 | grep -e _sqreen_hook_prolog_var -e _sqreen_hook_descriptor -e _sqreen_hook_table)
echo $size
