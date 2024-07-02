# ExternalDNS - Midaas-webhook

ExternalDNS is a Kubernetes add-on for automatically managing Domain Name System (DNS) records on Kubernetes objects (ingress, crd, services) by using different DNS providers (aws, ovh, webhook...). 

This repository use the [webhook provider](https://github.com/kubernetes-sigs/external-dns/blob/master/docs/tutorials/webhook-provider.md). This webhook is a sidecar running in the same pod as external-dns, which manage MiDaas dns records . 

To use ExternalDNS with MiDaas, you need to inject TSIG on each zone you want to manage.

## Kubernetes deployment

You can deploy the webhook using the following commands:

```
helm repo add external-dns https://kubernetes-sigs.github.io/external-dns/
```

1. Create the helm values file `external-dns-midaas-values.yaml`

```yaml
# if midaas can delete records in dns zone
policy: sync
# midaas manage all records on zone
registry: noop
# can restrict zone
domainFilters: ["subzone.dev.example.com"]
provider: 
  name: webhook
  webhook: 
    image: ghcr.io/titigmr/external-dns-midaas-webhook
    tag: v1.0.0
  env:
  - name: PROVIDER_SKIP_TLS_VERIFY
    value: "true"
  - name: PROVIDER_DNS_ZONE_SUFFIX
    value: "dev.example.com"
  - name: PROVIDER_WS_URL
    value: https://midaas.com/midaas/ws"
```

2. Create helm deployment

`helm install external-dns external-dns -f external-dns-midaas-values.yaml`

## Local development