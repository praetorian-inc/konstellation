#!/bin/sh

# This standalone bash script to enumerate kubernetes resources.
# This script requires jq and zip in $PATH.

OUTPUT="k8s-enum"
CWD=$(pwd)

mkdir $OUTPUT
cd $OUTPUT

for i in $(kubectl api-resources -o name); do kubectl get $i -o json > "${i}.json"; done

# remove the raw secret data
jq '.items | map(del(.data))'< secrets.json > secrets2.json
mv secrets2.json secrets.json
cd $CWD

zip -r ${OUTPUT}.zip $OUTPUT
