#!/bin/sh -e

TOKEN=gsFXkJqGrUNoYMQPZe4k3WKwijnrp8iGSwn3bApe

if [ $# -ne 2 ]; then
  echo "Usage: $0 <id> <country>" >&2
  exit 2
fi

id=$1
country=$2

fetch() {
  suffix=$1
  url="https://api.tidal.com/v1/albums/${id}${suffix}?countryCode=${country}"
  curl --silent "$url" -H "x-tidal-token:${TOKEN}"
}

fetch ''       >"album_${id}_${country}.json"
fetch /credits >"credits_${id}_${country}.json"
fetch /tracks  >"tracks_${id}_${country}.json"
