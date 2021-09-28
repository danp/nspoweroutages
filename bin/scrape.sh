#!/bin/bash

set -eo pipefail

mkdir -p data

go run ./cmd/scrape-outages | jq -S 'sort_by(.id)' > data/outages.json

dir="$(curl --silent --fail http://outagemap.nspower.ca/resources/data/external/interval_generation_data/metadata.json | jq -r .directory)"
durl="http://outagemap.nspower.ca/resources/data/external/interval_generation_data/$dir"

f="report_servicearea"
curl --silent --fail --retry 3 --max-time 15 -o - "$durl/$f.json" | jq -S . > "data/$f.json"

go run ./cmd/scrape-load | jq -S . > data/load.json
