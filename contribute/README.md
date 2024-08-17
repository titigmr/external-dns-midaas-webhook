# Run all stack locally with docker

## Kind cluster

One single node is deployed but it can be customized in `./kind/kind-config.yml`. The cluster comes with [Traefik](https://doc.traefik.io/traefik/providers/kubernetes-ingress/) or [Nginx](https://kind.sigs.k8s.io/docs/user/ingress/#ingress-nginx) ingress controller installed with port mapping on both ports `8080` and `8443`.

The node is using `extraMounts` to provide a volume binding between host working directory and `/app` to give the ability to bind mount volumes into containers during development.


## Midaas Webservice 

A wrapper of midaas is available for development on folder `./midaas-ws`. Note that tool not really do dns records. It only writes fake domain on container filesystem. 

This webservice is written in python with `Fastapi` framework. The webservice listen on 3 endpoints:
- `GET` - `/ws/{domaine}` : retrieve all domains for a specific zone
- `PUT` - `/ws/{domaine}/{type}/{valeur}` : add or modify a DNS record
You must add this body in the request: 
```json
{"ttl": 0, "keyname": "string", "keyvalue": "string"}
```
- `DELETE` - `/ws/{domaine}/{type}/{valeur}` : add or modify a DNS
You must add this body in the request:  
```json
{"keyname": "string", "keyvalue": "string"}
```

The midaas webservice can be configured with the following environment variables:

| Name            | Description            | Default   |
| --------------- | ---------------------- | --------- |
| MIDAAS_KEYNAME  | TSIG Keyname           | test      |
| MIDAAS_KEYVALUE | TSIG Keyvalue          | test      |
| MIDAAS_ZONE     | Zone managed by MiDaas | dev.local |


## External-DNS Locally

:construction: