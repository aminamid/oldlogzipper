#!/bin/bash

srcdir="$1"
dstdir="$2"

if [ -z "$srcdir" ] || [ -z "$dstdir" ]; then
    echo "Usage: movegz <srcdir> <dstdir>"
    exit 1
fi

timestamp=$(date +"%Y%m%d")
new_dstdir="$dstdir/$timestamp"

mkdir -p "$new_dstdir"

for file in "$srcdir"/*.gz; do
    [ -e "$file" ] || continue

    duplicate_count=0
    base_file_name=$(basename "$file")
    destination_file_path="$new_dstdir/$base_file_name"

    while [ -e "$destination_file_path" ]; do
        src_checksum=$(md5sum "$file" | awk '{print $1}')
        dst_checksum=$(md5sum "$destination_file_path" | awk '{print $1}')

        if [ "$src_checksum" == "$dst_checksum" ]; then
            echo "Skipping duplicate file: $file"
            break
        else
            duplicate_count=$((duplicate_count + 1))
            destination_file_path="$new_dstdir/${base_file_name%.gz}.duplicated${duplicate_count}.gz"
        fi
    done

    if [ ! -e "$destination_file_path" ]; then
        if [ $duplicate_count -gt 0 ]; then
            echo "Found different file with the same name. Renamed to: $(basename "$destination_file_path")"
        fi
        mv "$file" "$destination_file_path"
    fi
done

echo "Moved .gz files to $new_dstdir"

