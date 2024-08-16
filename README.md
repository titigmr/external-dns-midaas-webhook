# ExternalDNS - Midaas-webhook

ExternalDNS is a Kubernetes add-on for automatically managing Domain Name System (DNS) records on Kubernetes objects (ingress, crd, services) by using different DNS providers (aws, ovh, webhook...). 

This repository use the [webhook provider](https://github.com/kubernetes-sigs/external-dns/blob/master/docs/tutorials/webhook-provider.md). This webhook is a sidecar running in the same pod as external-dns, which manage MiDaas dns records. 

To use ExternalDNS with MiDaas, you need to inject TSIG on each zone you want to manage.

## Kubernetes deployment

You can deploy the webhook using the following commands:

```sh
helm repo add external-dns https://kubernetes-sigs.github.io/external-dns/
```

1. Create the helm values file `external-dns-midaas-values.yaml`:

```yaml
# -- How DNS records are synchronized between sources and providers; available values are `sync` & `upsert-only`.
policy: sync
# -- Specify the registry for storing ownership and labels.
# Valid values are `txt`, `aws-sd`, `dynamodb` & `noop`.
# If `noop` midaas manage all records on zone
registry: txt
# can restrict zone
domainFilters: ["subzone.d1.dev.example.com"]
provider: 
  name: webhook
  webhook: 
    image: ghcr.io/titigmr/external-dns-midaas-webhook
    tag: latest
  env:
  - name: PROVIDER_DNS_ZONE_SUFFIX
    value: "dev.example.com"
  - name: PROVIDER_WS_URL
    value: https://midaas.example.com/midaas/ws"
  - name: TSIG_ZONE_<TSIG_Keyname>
    value: <TSIG_Keyvalue>
```

2. Create helm deployment:

```sh
helm install external-dns external-dns -f external-dns-midaas-values.yaml
```

## Parameters references

| Name                     | Description                                               | Default value                             |
| ------------------------ | --------------------------------------------------------- | ----------------------------------------- |
| API_SERVER_PORT          | define the host where api listen, for all interfaces      | `"0.0.0.0"`                               |
| API_SERVER_HOST          | define the port where api listen                          | `"8888"`                                  |
| API_READ_TIMEOUT         | timout until read                                         | `3s`                                      |
| API_WRITE_TIMEOUT        | timeout until write                                       | `3s`                                      |
| API_LOG_LEVEL            | log level among `DEBUG`,`INFO`,`TRACE`,`WARN`,`ERROR`     | `INFO`                                    |
| PROVIDER_SKIP_TLS_VERIFY | enable tls verification                                   | `false`                                   |
| PROVIDER_DNS_ZONE_SUFFIX | dns zone suffix                                           | `"dev.example.com"`                       |
| PROVIDER_WS_URL          | webservice url                                            | `"https://midaas.example.com/midaas/ws/"` |
| TSIG_ZONE_<TSIG_Keyname> | tsigs credentials for manipulating one or multiples zones |                                           |

For example, `TSIG_ZONE_d1` with `PROVIDER_DNS_ZONE_SUFFIX` with `dev.example.com` refer to the folowing zone: `d1.dev.example.com`


## Local development

🚧 Work in progress.

## Contributions

Commits must follow the specification of [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/), it is possible to add the [VSCode extension](https://github.com/vivaxy/vscode-conventional-commits) to facilitate the creation of commits.

A PR must be made with an updated branch with the `main` branch in rebase (and without merge) before requesting a merge, and the merge must be requested in `main`.
