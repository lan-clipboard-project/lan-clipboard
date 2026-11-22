#!/bin/sh
# Read module name and package directory pairs separated by tabs.
set -eu

destination=$1
tab=$(printf '\t')
mkdir -p "$destination"
while IFS="$tab" read -r name directory; do
    [ -n "$name" ] || continue
    for notice in "$directory"/LICENSE* "$directory"/LICENCE* "$directory"/NOTICE* \
        "$directory"/COPYING* "$directory"/COPYRIGHT* "$directory"/license*; do
        [ -f "$notice" ] || continue
        mkdir -p "$destination/$name"
        cp "$notice" "$destination/$name/"
    done
done
