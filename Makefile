PROMPT_COLOR=\033[36m
PROMPT_NC=\033[0m # No Color

HELM_CHARTS_PATH = ../helm-charts/otterize-kubernetes

OTRZ_NAMESPACE = otterize-system
OTRZ_IMAGE_TAG = 0.0.0
OTRZ_IMAGE_REGISTRY = otterize
OTRZ_AGENT_IMAGE_NAME = agent
OTRZ_MAPPER_IMAGE_NAME = mapper
OTRZ_AGENT_IMAGE_FULL_NAME = $(OTRZ_IMAGE_REGISTRY)/$(OTRZ_AGENT_IMAGE_NAME):$(OTRZ_IMAGE_TAG)
OTRZ_MAPPER_IMAGE_FULL_NAME = $(OTRZ_IMAGE_REGISTRY)/$(OTRZ_MAPPER_IMAGE_NAME):$(OTRZ_IMAGE_TAG)

LIMA_K8S_TEMPLATE = ./dev/lima-k8s.yaml
LIMA_CLUSTER_NAME = k8s
LIMA_KUBECONFIG_PATH = $(HOME)/.kube/lima
LIMA_TEMP_DIR = /tmp/lima/
DOCKER_TARGET_ARCH = arm64
LIMA_TARGET_ARCH = aarch64

# Include .env file if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

help: ## Show help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n"} /^[$$()% a-zA-Z_-]+:.*?##/ { printf "  ${PROMPT_COLOR}%-25s${PROMPT_NC} %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

# Image building targets

build-agent: ## Builds the node agent image
	@echo "${PROMPT_COLOR}Building agent image '$(OTRZ_AGENT_IMAGE_FULL_NAME)'...${PROMPT_NC}"
	docker buildx build --platform linux/$(DOCKER_TARGET_ARCH) -t $(OTRZ_AGENT_IMAGE_FULL_NAME) --file build/$(OTRZ_AGENT_IMAGE_NAME).Dockerfile src/

build-mapper: ## Builds the mapper image
	@echo "${PROMPT_COLOR}Building mapper image '$(OTRZ_MAPPER_IMAGE_FULL_NAME)'...${PROMPT_NC}"
	docker buildx build --platform linux/$(DOCKER_TARGET_ARCH) -t $(OTRZ_MAPPER_IMAGE_FULL_NAME) --file build/$(OTRZ_MAPPER_IMAGE_NAME).Dockerfile src/

# Lima-specific targets - used for local development on macOS

lima-install: ## Installs lima if not already installed
	@echo "${PROMPT_COLOR}Installing Lima...${PROMPT_NC}"
	brew list lima || brew install lima

lima-k8s: ## Starts Lima with k8s template
	@echo "${PROMPT_COLOR}Starting Lima with the template '$(LIMA_K8S_TEMPLATE)'...${PROMPT_NC}"
	limactl start $(LIMA_K8S_TEMPLATE) --arch $(LIMA_TARGET_ARCH) --name k8s

lima-kubeconfig: ## Copies kubeconfig from lima to host
	@echo "${PROMPT_COLOR}Copying kubeconfig from Lima to host...${PROMPT_NC}"
	cp $(shell limactl list k8s --format '{{.Dir}}/copied-from-guest/kubeconfig.yaml') $(LIMA_KUBECONFIG_PATH)
	@echo "${PROMPT_COLOR}Run 'export KUBECONFIG=$(LIMA_KUBECONFIG_PATH)' to use the kubeconfig.${PROMPT_NC}"

lima-copy-images: ## Copies the images to lima
	@echo "${PROMPT_COLOR}Copying images to Lima...${PROMPT_NC}"
	docker save -o $(LIMA_TEMP_DIR)images/$(OTRZ_AGENT_IMAGE_NAME).tar $(OTRZ_AGENT_IMAGE_FULL_NAME)
	docker save -o $(LIMA_TEMP_DIR)images/$(OTRZ_MAPPER_IMAGE_NAME).tar $(OTRZ_MAPPER_IMAGE_FULL_NAME)

	limactl copy $(LIMA_TEMP_DIR)images/$(OTRZ_AGENT_IMAGE_NAME).tar k8s:/tmp/$(OTRZ_AGENT_IMAGE_NAME).tar
	limactl copy $(LIMA_TEMP_DIR)images/$(OTRZ_MAPPER_IMAGE_NAME).tar k8s:/tmp/$(OTRZ_MAPPER_IMAGE_NAME).tar

	LIMA_INSTANCE=$(LIMA_CLUSTER_NAME) lima sudo ctr -n=k8s.io images import /tmp/$(OTRZ_AGENT_IMAGE_NAME).tar
	LIMA_INSTANCE=$(LIMA_CLUSTER_NAME) lima sudo ctr -n=k8s.io images import /tmp/$(OTRZ_MAPPER_IMAGE_NAME).tar

lima-restart-otterize: ## Restarts Otterize pods running in the lima kubernetes cluster
	@echo "${PROMPT_COLOR}Restarting Otterize pods...${PROMPT_NC}"
	LIMA_INSTANCE=$(LIMA_CLUSTER_NAME) lima kubectl delete pods --all -n $(OTRZ_NAMESPACE)

lima-update-images: build-mapper build-agent lima-copy-images lima-restart-otterize ## Builds and updates the mapper image in the lima kubernetes cluster and restarts the pods

lima-install-otterize: ## Installs Otterize in the lima kubernetes cluster with the provided client ID and client secret
	@if [ -z "$(CLIENT_ID)" ]; then \
	  read -p "Client ID: " client_id; \
	else \
	  client_id=$(CLIENT_ID); \
	fi; \
	if [ -z "$(CLIENT_SECRET)" ]; then \
	  read -p "Client Secret: " client_secret; \
	else \
	  client_secret=$(CLIENT_SECRET); \
	fi; \
    helm --kubeconfig=$(LIMA_KUBECONFIG_PATH) dep up ../helm-charts/otterize-kubernetes; \
    helm --kubeconfig=$(LIMA_KUBECONFIG_PATH) upgrade --install \
    	otterize $(HELM_CHARTS_PATH) -n $(OTRZ_NAMESPACE) --create-namespace \
		--set networkMapper.sniffer.enable=false \
		--set networkMapper.debug=true \
		--set networkMapper.agent.tag=$(OTRZ_IMAGE_TAG) \
		--set networkMapper.agent.image=$(OTRZ_AGENT_IMAGE_NAME) \
		--set networkMapper.agent.pullPolicy=Never \
		--set networkMapper.bpfman.tag=$(OTRZ_IMAGE_TAG) \
		--set networkMapper.bpfman.image=$(OTRZ_BPFMAN_IMAGE_NAME) \
		--set networkMapper.bpfman.pullPolicy=Never \
		--set networkMapper.mapper.tag=$(OTRZ_IMAGE_TAG) \
		--set networkMapper.mapper.image=$(OTRZ_MAPPER_IMAGE_NAME) \
		--set networkMapper.mapper.pullPolicy=Never \
		--set intentsOperator.operator.mode=defaultShadow \
		--set global.otterizeCloud.apiAddress=http://host.lima.internal:3000/api \
		--set global.otterizeCloud.credentials.clientId=$$client_id \
		--set global.otterizeCloud.credentials.clientSecret=$$client_secret


setup-lima: lima-install lima-k8s lima-kubeconfig lima-update-images lima-install-otterize ## Setup Lima with kubernetes template
	@echo "${PROMPT_COLOR}Setup completed.${PROMPT_NC}"
	LIMA_INSTANCE=$(LIMA_CLUSTER_NAME) lima kubectl get pods -n otterize-system

clean-lima: ## Cleans up lima environment
	@echo "${PROMPT_COLOR}Cleaning up '$(LIMA_K8S_TEMPLATE)'...${PROMPT_NC}"
	limactl stop -f $(LIMA_CLUSTER_NAME)
	limactl delete $(LIMA_CLUSTER_NAME)