MIDAAS_IMAGE := api-midaas:v1
WEBHOOK_IMAGE := webhook:v1
ENV_MIDAAS_KEYNAME := test
ENV_MIDAAS_KEYVALUE := test
ENV_MIDAAS_ZONES := dev.local
CACHE_CONFIG := /tmp/cluster-name.txt
CLUSTER_NAME := go

build-midaas:
	docker build -t ${MIDAAS_IMAGE} midaas-ws/ 

push-midaas:
	kind load docker-image ${MIDAAS_IMAGE} --name ${CLUSTER_NAME}

delete-midaas:
	@kubectl delete pod midaas --ignore-not-found
	@kubectl delete svc midaas --ignore-not-found

deploy-midaas: delete-midaas
	@kubectl run --image ${MIDAAS_IMAGE} --expose=true --port 8080 \
		--env "MIDAAS_KEYNAME=${ENV_MIDAAS_KEYNAME}" \
		--env "MIDAAS_KEYVALUE=${ENV_MIDAAS_KEYVALUE}" \
		--env "MIDAAS_ZONES=${ENV_MIDAAS_ZONES}" midaas
	@echo "\n\n-------------------"
	@echo "Kubernetes midaas service is listening on port 8080"
	@echo "Use 'TSIG_${ENV_MIDAAS_KEYNAME}=${ENV_MIDAAS_KEYVALUE}' on environment variable in webhook"
	@echo "For manual actions, OpenAPI schema is available at path /docs" 


start-midaas: build-midaas push-midaas deploy-midaas

