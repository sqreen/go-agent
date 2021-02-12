#!/usr/bin/env bash

declare -A memsizes
while read -r segment; do
    cols=( $segment )
    memsize=${cols[5]}
    len=${#cols[@]}
    memflags=( ${cols[@]:6:$len-6-1} )
    let id
    case "${memflags[@]}" in
        "R")
        id=rodata
        ;;
        "RW")
        id=data
        ;;
        "R E")
        id=text
        ;;
        *)
        id=unknown
    esac

    # store in the map and convert the hexadecimal value into the decimal value
    # using the bash calculation which converts it automatically.
    memsizes[$id]=$(($memsize))
done < <(readelf -l -W $1 | grep LOAD)

echo ${!memsizes[@]}
echo ${memsizes[@]}