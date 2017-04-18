#!/bin/bash

ttlify() {
  local i
  for i in "$@"; do
    [[ "${i}" =~ ^([0-9]+)([a-z]*)$ ]] || continue
    local num="${BASH_REMATCH[1]}"
    local unit="${BASH_REMATCH[2]}"
    case "${unit}" in
                     weeks|week|wee|we|w) unit=''; num=$[num*60*60*24*7];;
                           days|day|da|d) unit=''; num=$[num*60*60*24];;
                     hours|hour|hou|ho|h) unit=''; num=$[num*60*60];;
      minutes|minute|minut|minu|min|mi|m) unit=''; num=$[num*60];;
      seconds|second|secon|seco|sec|se|s) unit=''; num=$[num];;
    esac
    echo "${num}${unit}"
  done
}

dns_start() {
  gcloud dns record-sets transaction start    -z "${ZONENAME}" --project "${PROJECT}"
}

dns_info() {
  gcloud dns record-sets transaction describe -z "${ZONENAME}" --project "${PROJECT}"
}

dns_abort() {
  gcloud dns record-sets transaction abort    -z "${ZONENAME}" --project "${PROJECT}"
}

dns_commit() {
  gcloud dns record-sets transaction execute  -z "${ZONENAME}" --project "${PROJECT}"
}

dns_add() {
  if [[ -n "$1" && "$1" != '@' ]]; then
    local -r name="$1.${ZONE}."
  else
    local -r name="${ZONE}."
  fi
  local -r ttl="$(ttlify "$2")"
  local -r type="$3"
  shift 3
  gcloud dns record-sets transaction add      -z "${ZONENAME}" --name "${name}" --ttl "${ttl}" --type "${type}" "$@" --project "${PROJECT}"
}

dns_del() {
  if [[ -n "$1" && "$1" != '@' ]]; then
    local -r name="$1.${ZONE}."
  else
    local -r name="${ZONE}."
  fi
  local -r ttl="$(ttlify "$2")"
  local -r type="$3"
  shift 3
  gcloud dns record-sets transaction remove   -z "${ZONENAME}" --name "${name}" --ttl "${ttl}" --type "${type}" "$@" --project "${PROJECT}"
}

lookup_dns_ip() {
  host "$1" | sed -rn 's@^.* has address @@p'
}

my_ip() {
  curl "http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip" -H "Metadata-Flavor: Google"
}

doit() {
  PROJECT=stone-swallow
  ZONE=sinmetal.org
  ZONENAME=sinmetal-org
  dns_start
  dns_del $HOSTNAME 5min A `lookup_dns_ip "$HOSTNAME.${ZONE}."`
  dns_add $HOSTNAME 5min A `my_ip`
  dns_commit
}

doit
