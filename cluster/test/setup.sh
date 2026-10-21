#!/usr/bin/env bash
set -aeuo pipefail

echo "Running setup.sh"
echo "Creating cloud credential secret..."
${KUBECTL} -n upbound-system create secret generic provider-secret --from-literal=credentials="${UPTEST_CLOUD_CREDENTIALS}" --dry-run=client -o yaml | ${KUBECTL} apply -f -

echo "Waiting until provider is healthy..."
${KUBECTL} wait provider.pkg --all --for condition=Healthy --timeout 5m

echo "Waiting for all pods to come online..."
${KUBECTL} -n upbound-system wait --for=condition=Available deployment --all --timeout=5m

echo "Creating a default provider config..."
cat <<YAML | ${KUBECTL} apply -f -
apiVersion: coreweave.upbound.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: provider-secret
      namespace: upbound-system
      key: credentials
YAML

echo "Creating a default cluster provider config (v2-style)..."
cat <<YAML | ${KUBECTL} apply -f -
apiVersion: coreweave.m.upbound.io/v1beta1
kind: ClusterProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: provider-secret
      namespace: upbound-system
      key: credentials
YAML

${KUBECTL} wait provider.pkg --all --for condition=Healthy --timeout 5m
${KUBECTL} -n upbound-system wait --for=condition=Available deployment --all --timeout=5m
