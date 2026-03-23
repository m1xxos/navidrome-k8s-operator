#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-navidrome-op}"
NAMESPACE="${NAMESPACE:-navidrome-operator}"
IMAGE_REPO="${IMAGE_REPO:-navidrome-operator}"
IMAGE_TAG="${IMAGE_TAG:-dev}"

if ! command -v kind >/dev/null 2>&1; then
  echo "kind is not installed. Install: brew install kind"
  exit 1
fi
if ! command -v helm >/dev/null 2>&1; then
  echo "helm is not installed. Install: brew install helm"
  exit 1
fi
if ! command -v kubectl >/dev/null 2>&1; then
  echo "kubectl is not installed. Install: brew install kubectl"
  exit 1
fi
if ! command -v docker >/dev/null 2>&1; then
  echo "docker is not installed. Install Docker Desktop"
  exit 1
fi

echo "==> Checking kind cluster ${CLUSTER_NAME}"
if ! kind get clusters | grep -qx "${CLUSTER_NAME}"; then
  kind create cluster --name "${CLUSTER_NAME}"
else
  echo "Cluster already exists"
fi

echo "==> Building local image ${IMAGE_REPO}:${IMAGE_TAG}"
docker build -t "${IMAGE_REPO}:${IMAGE_TAG}" .

echo "==> Loading image into kind"
kind load docker-image "${IMAGE_REPO}:${IMAGE_TAG}" --name "${CLUSTER_NAME}"

echo "==> Applying CRDs"
kubectl apply -f config/crd/bases

echo "==> Installing operator chart"
helm upgrade --install navidrome-operator ./charts/navidrome-operator \
  -n "${NAMESPACE}" \
  --create-namespace \
  --skip-crds \
  --set image.repository="${IMAGE_REPO}" \
  --set image.tag="${IMAGE_TAG}" \
  --set image.pullPolicy=IfNotPresent

echo "==> Applying sample resources"
if [[ -f config/samples/secret.local.yaml ]]; then
  kubectl apply -f config/samples/secret.local.yaml
else
  kubectl apply -f config/samples/secret.yaml
fi
kubectl apply -f config/samples/playlist.yaml

if [[ -f config/samples/tracks.coding.yaml ]]; then
  kubectl delete track -n default -l navidrome.m1xxos.dev/managed-by=generated --ignore-not-found
  kubectl apply -f config/samples/tracks.coding.yaml
else
  kubectl apply -f config/samples/track.yaml
fi

echo
echo "All set. Useful commands:"
echo "  kubectl get playlists,tracks -A"
echo "  kubectl get pods -n ${NAMESPACE}"
echo "  kubectl logs -n ${NAMESPACE} deploy/navidrome-operator-navidrome-operator -f"
