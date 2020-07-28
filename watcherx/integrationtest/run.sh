#!/bin/bash
set -euxo pipefail

CLUSTER_NAME=watcherx-integration-test

docker build -f Dockerfile -t eventlogger:latest ../..
kind create cluster --name $CLUSTER_NAME --wait 1m || true
kind load docker-image eventlogger:latest --name $CLUSTER_NAME
kubectl apply -f configmap.yml -f event_logger.yml --context kind-$CLUSTER_NAME
