#!/bin/bash

mkdir -p internal/neofs/services/tree 2>/dev/null

REVISION="feaa9eace7098c343598bf08fb50746a1e8d2deb"

echo "tree service revision ${REVISION}"

FILES=$(curl -s https://github.com/TrueCloudLab/frostfs-node/tree/${REVISION}/pkg/services/tree | sed -n "s,.*\"/TrueCloudLab/frostfs-node/blob/${REVISION}/pkg/services/tree/\(.*\.pb\.go\)\".*,\1,p")

for file in $FILES; do
  if [[ $file == *"neofs"* ]]; then
    echo "skip '$file'"
    continue
  else
    echo "sync '$file' in tree service"
  fi
  curl -s "https://raw.githubusercontent.com/TrueCloudLab/frostfs-node/${REVISION}/pkg/services/tree/${file}" -o "./internal/neofs/services/tree/${file}"
done
