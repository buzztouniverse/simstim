# Provider scripts

A provider script lets the client start and stop a host, so a cloud VM can sleep when you are not using it. The client knows nothing about any cloud: the script does the work.

## Contract

The client runs `script <action>`, with a JSON job on stdin:

```json
{"action": "start", "host": {"name": "gcp-dev", "ssh": "gcp-dev", "labels": {"zone": "europe-west2-b"}}}
```

| Action | Does | Prints |
| --- | --- | --- |
| `start` | Start the VM. Return when the provider says it is running. | Anything, or nothing |
| `stop` | Stop the VM. Its sessions stop too. | Anything, or nothing |
| `status` | Ask whether the VM runs. | `{"state": "running"}` or `{"state": "stopped"}` |

A non-zero exit is an error, and its stderr is shown to the user. Labels from `hosts.toml` carry the provider's own names (project, zone, instance ID), so one script serves many hosts.

## Use

```sh
<client> host add gcp-dev --provider-script ~/bin/gcp-provider.sh --wake --label zone=europe-west2-b
<client> host stop gcp-dev     # at the end of the day
<client> attach gcp-dev/api    # starts the VM first, because of --wake
```

## Examples

| Script | Provider |
| --- | --- |
| [gcp.sh](gcp.sh) | Google Compute Engine, with `gcloud` |
| [aws.sh](aws.sh) | Amazon EC2, with the `aws` CLI |

They are starting points. Read them before you use them.
