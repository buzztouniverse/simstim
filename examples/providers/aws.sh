#!/bin/sh
# Provider script for an Amazon EC2 instance, using the aws CLI.
# Labels in hosts.toml: instance_id (required), region.
set -eu
job=$(cat)
label() { printf '%s' "$job" | python3 -c "import json,sys; print(json.load(sys.stdin)['host'].get('labels',{}).get('$1',''))"; }
id=$(label instance_id)
region=$(label region)
[ -n "$id" ] || { echo "set the label instance_id for this host" >&2; exit 2; }
set -- "$1" ${region:+--region "$region"}
case "$1" in
  start)  shift; aws ec2 start-instances --instance-ids "$id" "$@" >/dev/null
          aws ec2 wait instance-running --instance-ids "$id" "$@" ;;
  stop)   shift; aws ec2 stop-instances --instance-ids "$id" "$@" >/dev/null ;;
  status) shift
          s=$(aws ec2 describe-instances --instance-ids "$id" "$@" --query 'Reservations[0].Instances[0].State.Name' --output text)
          [ "$s" = running ] && echo '{"state":"running"}' || echo '{"state":"stopped"}' ;;
  *) echo "unknown action $1" >&2; exit 2 ;;
esac
