#!/bin/sh
# Provider script for a Google Compute Engine VM, using gcloud.
# Labels in hosts.toml: instance (default: the host's name), zone, project.
set -eu
job=$(cat)
label() { printf '%s' "$job" | python3 -c "import json,sys; print(json.load(sys.stdin)['host'].get('labels',{}).get('$1',''))"; }
name=$(printf '%s' "$job" | python3 -c "import json,sys; print(json.load(sys.stdin)['host']['name'])")
instance=$(label instance); instance=${instance:-$name}
zone=$(label zone)
project=$(label project)
set -- "$1" --zone "$zone" ${project:+--project "$project"}
case "$1" in
  start)  shift; gcloud compute instances start "$instance" "$@" --quiet ;;
  stop)   shift; gcloud compute instances stop "$instance" "$@" --quiet ;;
  status) shift
          s=$(gcloud compute instances describe "$instance" "$@" --format='value(status)')
          [ "$s" = RUNNING ] && echo '{"state":"running"}' || echo '{"state":"stopped"}' ;;
  *) echo "unknown action $1" >&2; exit 2 ;;
esac
