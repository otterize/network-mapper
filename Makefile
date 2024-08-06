PROMPT_COLOR=\033[36m
PROMPT_NC=\033[0m # No Color

OTRZ_NAMESPACE = otterize-system
OTRZ_MAPPER_IMAGE_NAME = otterize/mapper:0.0.0

LIMA_K8S_TEMPLATE = template://k8s
LIMA_CLUSTER_NAME = k8s
LIMA_KUBECONFIG_PATH = ~/.kube/lima
LIMA_TEMP_DIR = /tmp/lima/

help: ## Show help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n"} /^[$$()% a-zA-Z_-]+:.*?##/ { printf "  ${PROMPT_COLOR}%-25s${PROMPT_NC} %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

# Image building targets

build-mapper: ## Builds the mapper image
	@echo "${PROMPT_COLOR}Building mapper image '$(OTRZ_MAPPER_IMAGE_NAME)'...${PROMPT_NC}"
	docker buildx build --platform linux/arm64 -t $(OTRZ_MAPPER_IMAGE_NAME) --file build/mapper.Dockerfile src/

# Lima-specific targets - used for local development on macOS

lima-install: ## Installs lima if not already installed
	@echo "${PROMPT_COLOR}Installing Lima...${PROMPT_NC}"
	brew list lima || brew install lima

lima-k8s: ## Starts Lima with k8s template
	@echo "${PROMPT_COLOR}Starting Lima with the template '$(LIMA_K8S_TEMPLATE)'...${PROMPT_NC}"
	limactl start $(LIMA_K8S_TEMPLATE)

lima-kubeconfig: ## Copies kubeconfig from lima to host
	@echo "${PROMPT_COLOR}Copying kubeconfig from Lima to host...${PROMPT_NC}"
	cp $(shell limactl list k8s --format '{{.Dir}}/copied-from-guest/kubeconfig.yaml') $(LIMA_KUBECONFIG_PATH)
	@echo "${PROMPT_COLOR}Run 'export KUBECONFIG=$(LIMA_KUBECONFIG_PATH)' to use the kubeconfig.${PROMPT_NC}"

lima-copy-images: ## Copies the images to lima
	@echo "${PROMPT_COLOR}Copying images to Lima...${PROMPT_NC}"
	docker save -o $(LIMA_TEMP_DIR)images/mapper.tar $(OTRZ_MAPPER_IMAGE_NAME)

	limactl copy $(LIMA_TEMP_DIR)images/mapper.tar k8s:/tmp/mapper.tar
	LIMA_INSTANCE=$(LIMA_CLUSTER_NAME) && lima sudo ctr -n=k8s.io images import /tmp/mapper.tar

lima-restart-otterize: ## Restarts Otterize pods running in the lima kubernetes cluster
	@echo "${PROMPT_COLOR}Restarting Otterize pods...${PROMPT_NC}"
	LIMA_INSTANCE=$(LIMA_CLUSTER_NAME) && lima kubectl delete pods --all -n $(OTRZ_NAMESPACE)

lima-update-mapper: build-mapper lima-copy-images lima-restart-otterize ## Updates the mapper image in the lima kubernetes cluster

setup-lima: lima-install lima-k8s lima-kubeconfig ## Setup Lima with kubernetes template
	@echo "${PROMPT_COLOR}Setup completed. You can now install Otterize in the new cluster${PROMPT_NC}"
	lima kubectl get pods -n otterize-system

clean-lima: ## Cleans up lima environment
	@echo "${PROMPT_COLOR}Cleaning up '$(LIMA_K8S_TEMPLATE)'...${PROMPT_NC}"
	limactl stop -f $(LIMA_CLUSTER_NAME)
	limactl delete $(LIMA_CLUSTER_NAME)