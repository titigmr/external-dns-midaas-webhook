SHELL := /bin/bash
DOCKER	:= $(shell type -p docker)
KIND	:= $(shell type -p kind)
HELM	:= $(shell type -p helm)
KUBECTL	:= $(shell type -p helm)

WEBHOOK_IMAGE := external-dns-midaas-webhook:dev
WEBHOOK_FOLDER := ./

MIDAAS_IMAGE := api-midaas:dev
MIDAAS_FOLDER := ./contribute/midaas-ws/

export MIDAAS_WS_URL ?= http://midaas.default:8080/ws/
export MIDAAS_DEV_SUFFIX ?= dev.local
export MIDAAS_ENV_KEYNAME ?= d1
export MIDAAS_ENV_KEYVALUE ?= test
export MIDAAS_ENV_ZONES ?= $(MIDAAS_ENV_KEYNAME).$(MIDAAS_DEV_SUFFIX)

KIND_CLUSTER_NAME ?= midaas
KIND_INGRESS_CONTROLLER = NGINX

# Commons targets

all: deploy-MIDAAS deploy-WEBHOOK create-test-ingress

clean: delete-cluster delete-image-MIDAAS delete-image-WEBHOOK

create-test-ingress: create-cluster	check-prerequisites-kubectl
	@kubectl apply -f ./contribute/ressources/ingress.yaml

logs-%:
	@kubectl logs -f  deployments/external-dns -c $*

midaas-get-zone:
	@kubectl exec midaas cat /tmp/$(MIDAAS_ENV_KEYNAME).$(MIDAAS_DEV_SUFFIX) | jq 

# Check prerequisites

check-prerequisites-docker:
ifeq ("$(wildcard ${DOCKER})","")
	@echo "docker not found" ; exit 1
endif
check-prerequisites-kind:
ifeq ("$(wildcard ${KIND})","")
	@echo "'kind' not found" ; exit 1
endif
check-prerequisites-kubectl:
ifeq ("$(wildcard ${KUBECTL})","")
	@echo "'kubectl' not found" ; exit 1
endif
check-prerequisites-helm:
ifeq ("$(wildcard ${HELM})","")
	@echo "'helm' not found" ; exit 1
endif

# Kind targets

create-cluster: check-prerequisites-kind
ifeq ($(shell kind get clusters |grep $(KIND_CLUSTER_NAME)), $(KIND_CLUSTER_NAME))
	@echo "Kind cluster '$(KIND_CLUSTER_NAME)' already exists, skipping"
else
	@kind create cluster --name $(KIND_CLUSTER_NAME) --config ./contribute/kind/kind-config.yaml
	@kubectl config use-context kind-$(KIND_CLUSTER_NAME)
endif

delete-cluster: check-prerequisites-kind
ifeq ($(shell kind get clusters |grep $(KIND_CLUSTER_NAME)), $(KIND_CLUSTER_NAME))
		@kind delete cluster --name $(KIND_CLUSTER_NAME)
endif

start-ingress-controller: create-cluster
ifeq ($(KIND_INGRESS_CONTROLLER), NGINX)
	@if [ ! -s /tmp/external-dns-nginx.yaml ]; then curl -Ls https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml > /tmp/external-dns-nginx.yaml; fi 
	@kubectl apply -f /tmp/external-dns-nginx.yaml
	@kubectl wait --namespace ingress-nginx \
  	--for=condition=ready pod \
  	--selector=app.kubernetes.io/component=controller \
  	--timeout=90s
else ifeq ($(KIND_INGRESS_CONTROLLER), TRAEFIK)
	@echo traefik
endif

# Docker build and push targets
build-%: check-prerequisites-docker
	@if [ $* = WEBHOOK ]; then docker build --target dev ${OPTIONS} -t  ${$*_IMAGE}  ${$*_FOLDER}; else docker build ${OPTIONS} -t  ${$*_IMAGE}  ${$*_FOLDER}; fi  

push-%: check-prerequisites-kind
	kind load docker-image ${$*_IMAGE} --name $(KIND_CLUSTER_NAME)

delete-image-%: check-prerequisites-docker
	docker rmi ${$*_IMAGE}

# Midaas deployment targets
delete-MIDAAS: create-cluster check-prerequisites-kubectl
	@kubectl delete pod midaas --ignore-not-found
	@kubectl delete svc midaas --ignore-not-found

deploy-MIDAAS: create-cluster check-prerequisites-kubectl build-MIDAAS push-MIDAAS delete-MIDAAS
	@kubectl run --image $(MIDAAS_IMAGE) --expose=true --port 8080 \
		--env "MIDAAS_KEYNAME=$(MIDAAS_ENV_KEYNAME)" \
		--env "MIDAAS_KEYVALUE=$(MIDAAS_ENV_KEYVALUE)" \
		--env "MIDAAS_ZONES=$(MIDAAS_ENV_ZONES)" midaas
	@echo "Kubernetes midaas service is listening on port 8080"


# Webhook deployment targets

delete-WEBHOOK: create-cluster check-prerequisites-helm
	@if [ "external-dns" == "$(shell helm ls -f external-dns -o json |jq -r .[].name)" ]; then helm delete external-dns; else echo "No external-dns release is currently running"; fi 

deploy-WEBHOOK: start-ingress-controller check-prerequisites-helm build-WEBHOOK push-WEBHOOK delete-WEBHOOK
	@echo "Adding repository"
	@helm repo add external-dns https://kubernetes-sigs.github.io/external-dns/
	@envsubst < ./contribute/ressources/external-dns-values.yaml > /tmp/external-dns-values.yaml
	@helm upgrade --force --install external-dns external-dns/external-dns -f /tmp/external-dns-values.yaml
	@echo "external DNS is running with webhook in sidecar"

