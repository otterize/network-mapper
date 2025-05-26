#!/bin/bash
#
#set -euo pipefail
#
## Required inputs (can also be passed as env vars)
#MAPPER_TAG="06f15b3088aee47db853d9c321a7b97efc6534c4"
#SNIFFER_TAG="06f15b3088aee47db853d9c321a7b97efc6534c4"
#MAPPER_IMAGE="mapper"
#SNIFFER_IMAGE="sniffer"
#REGISTRY="us-central1-docker.pkg.dev/main-383408/otterize"
#INSTALL_EXTRA_FLAGS=" --set kafkawatcher.enable=true --set kafkawatcher.kafkaServers={\"kafka-0.kafka\"} "
#
## Ensure required commands are available
#for cmd in docker kubectl helm minikube jq ; do
#  command -v $cmd >/dev/null 2>&1 || { echo >&2 "$cmd is required but not installed."; exit 1; }
#done
#
#echo ">> Starting minikube"
#minikube start --cpus=4 --memory 4096 --disk-size 32g --cni=calico
#
#sleep 10
#
#echo ">> Waiting for Calico to be ready"
#kubectl wait pods -n kube-system -l k8s-app=calico-kube-controllers --for condition=Ready --timeout=120s
#kubectl wait pods -n kube-system -l k8s-app=calico-node --for condition=Ready --timeout=120s
#
#echo ">> Pulling Docker images"
#docker pull "${REGISTRY}/mapper:${MAPPER_TAG}"
#minikube image load "${REGISTRY}/mapper:${MAPPER_TAG}"
#docker pull "${REGISTRY}/sniffer:${SNIFFER_TAG}"
#minikube image load "${REGISTRY}/sniffer:${SNIFFER_TAG}"
#
#echo ">> Setting up Helm dependencies and deploying Network Mapper"
#helm dep up ./helm-charts/network-mapper
#helm install otterize ./helm-charts/network-mapper -n otterize-system --create-namespace \
#  --set debug=true \
#  --set-string mapper.repository="${REGISTRY}" \
#  --set-string mapper.image="${MAPPER_IMAGE}" \
#  --set-string mapper.tag="${MAPPER_TAG}" \
#  --set-string mapper.pullPolicy=Never \
#  --set-string sniffer.repository="${REGISTRY}" \
#  --set-string sniffer.image="${SNIFFER_IMAGE}" \
#  --set-string sniffer.tag="${SNIFFER_TAG}" \
#  --set-string sniffer.pullPolicy=Never \
#  --set global.telemetry.enabled=false \
#  $INSTALL_EXTRA_FLAGS
#
#
#echo ">> Waiting for Otterize components to be ready"
#kubectl wait pods -n otterize-system -l app=otterize-network-sniffer --for condition=Ready --timeout=90s
#kubectl wait pods -n otterize-system -l app=otterize-network-mapper --for condition=Ready --timeout=90s
#
#echo ">> Deploying Kafka via Helm"
#helm repo add otterize https://helm.otterize.com
#helm repo update
#helm install --create-namespace -n kafka -f https://docs.otterize.com/code-examples/kafka-mapping/helm/values.yaml kafka otterize/kafka --version 21.4.4
#
#echo ">> Deploying Kafka Tutorial Services"
#kubectl apply -n otterize-tutorial-kafka-mapping -f https://docs.otterize.com/code-examples/kafka-mapping/all.yaml
#
#echo ">> Waiting for Kafka and Tutorial Services"
#sleep 10
#kubectl wait pods -n kafka -l app.kubernetes.io/component=kafka --for condition=Ready --timeout=180s
#kubectl wait pods -n kafka -l app.kubernetes.io/component=zookeeper --for condition=Ready --timeout=180s
#kubectl wait pods -n otterize-tutorial-kafka-mapping -l app=client --for condition=Ready --timeout=90s
#kubectl wait pods -n otterize-tutorial-kafka-mapping -l app=client-2 --for condition=Ready --timeout=90s
#
#
#echo ">> Waiting for intents to be discovered..."
#for i in {1..5}; do
#
#    OUTPUT_JSON=`otterize network-mapper export --telemetry-enabled=false -n otterize-tutorial-kafka-mapping --format=json`
#    if [ `echo "$OUTPUT_JSON" | jq ". | length"` != 2 ] || [ `echo "$OUTPUT_JSON" | jq '[.[] | select(.spec.targets[] | has("kafka"))] | length'` != 2 ] ; then
#      echo "wait for discovered intents";
#      echo _SNIFFER LOGS_
#      kubectl logs --since=15s -n otterize-system -l app=otterize-network-sniffer
#      echo _MAPPER LOGS_
#      kubectl logs --since=15s -n otterize-system -l app=otterize-network-mapper
#      sleep 10 ;
#    fi
#
##    intents_count=$(otterize network-mapper export --telemetry-enabled=false -n otterize-tutorial-kafka-mapping --format=json | jq length)
##    if [ "$intents_count" -eq 2 ]; then
##        echo "Intents discovered"
##        break
##    fi
##    echo "Waiting... ($i)"
##    echo "_SNIFFER LOGS_"
##    kubectl logs --since=15s -n otterize-system -l app=otterize-network-sniffer
##    echo "_MAPPER LOGS_"
##    kubectl logs --since=15s -n otterize-system -l app=otterize-network-mapper
##    sleep 10
#done
#
#echo ">> Outputting final logs"
#kubectl logs -n otterize-system -l app=otterize-network-sniffer --tail=-1
#kubectl logs -n otterize-system -l app=otterize-network-mapper --tail=-1
#
#echo ">> Exporting and comparing discovered intents"
#

echo "export intents and compare to expected file"
INTENTS_JSON=`otterize network-mapper export --telemetry-enabled=false -n otterize-tutorial-kafka-mapping --format=json`
echo "1"
echo $INTENTS_JSON
INTENTS_JSON_NO_KIND=`echo "$INTENTS_JSON" | jq 'map(del(.spec.workload.kind))'`
echo "2"
echo $INTENTS_JSON_NO_KIND
INTENTS_JSON_NO_KIND_AND_SORTED=`echo "$INTENTS_JSON_NO_KIND" | jq 'sort_by(.metadata.namespace + .metadata.name) | map(.spec.targets |= (sort_by(keys_unsorted[0]) | map(if .kafka? then .kafka.topics |= map(.operations |= sort) else . end)))'`
echo "3"
echo $INTENTS_JSON_NO_KIND_AND_SORTED
echo "$INTENTS_JSON_NO_KIND_AND_SORTED" > /tmp/intents.json
#echo "expected" && cat .github/workflows/tests-expected-results/kafka-tutorial-intents.json
#echo "actual" && cat /tmp/intents.json
#diff .github/workflows/tests-expected-results/kafka-tutorial-intents.json /tmp/intents.json

#
#otterize network-mapper export --telemetry-enabled=false -n otterize-tutorial-kafka-mapping --format=json | jq \
#'sort_by(.metadata.namespace, .metadata.name) | map(
#    .spec.targets |= (sort_by(keys[0])
#      | map(
#          if .kafka?
#          then .kafka.topics |= map(
#              .operations |= sort
#            )
#          else .
#          end
#        )
#    )
#  )' > /tmp/intents.json
#
#diff .github/workflows/tests-expected-results/kafka-tutorial-intents.json /tmp/intents.json && echo "Test passed" || {
#    echo "Test failed"
#    echo "Expected:"
#    cat .github/workflows/tests-expected-results/kafka-tutorial-intents.json
#    echo "Actual:"
#    cat /tmp/intents.json
#    exit 1
#}
