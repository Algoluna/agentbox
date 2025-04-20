#!/bin/bash

set -e

# --- Cleanup Target ---
if [[ "$1" == "clean" || "$1" == "cleanup" ]]; then
  echo "=== Cleaning up all AgentBox namespaces and build artifacts ==="
  kubectl delete namespace agentbox-system --ignore-not-found
  kubectl delete namespace agent-hello-agent --ignore-not-found
  rm -f operator.tar agent-debug.tar
  echo "Cleanup complete."
  exit 0
fi

# Define image tags
OPERATOR_IMAGE_BASE="agent-operator"
HELLO_AGENT_IMAGE_BASE="hello-agent"
HELLO_AGENT_IMAGE_TAG="debug-$(date +%Y%m%d-%H%M%S)"
OPERATOR_IMAGE_TAG="latest"
MICROK8S_REGISTRY="localhost:32000"
OPERATOR_IMAGE="${MICROK8S_REGISTRY}/${OPERATOR_IMAGE_BASE}:${OPERATOR_IMAGE_TAG}"
HELLO_AGENT_IMAGE="${MICROK8S_REGISTRY}/${HELLO_AGENT_IMAGE_BASE}:${HELLO_AGENT_IMAGE_TAG}"

NAMESPACE="agentbox-system"
RELEASE_NAME="agentbox"

echo "--- Setting up Prerequisites ---"
./scripts/setup_prereqs.sh

echo "--- Building Agent Operator Image ---"
cd agent-operator
make docker-build IMG=${OPERATOR_IMAGE_BASE}:${OPERATOR_IMAGE_TAG} > /dev/null 2>&1
cd ..
echo "Agent Operator Image Built: ${OPERATOR_IMAGE_BASE}:${OPERATOR_IMAGE_TAG}"

echo "--- Saving Agent Operator Image ---"
docker save ${OPERATOR_IMAGE_BASE}:${OPERATOR_IMAGE_TAG} -o operator.tar
echo "Agent Operator Image Saved to operator.tar"

echo "--- Importing Agent Operator Image into microk8s ---"
microk8s ctr image import operator.tar
echo "Agent Operator Image Imported into microk8s"
rm operator.tar

echo "--- Tagging Agent Operator Image in microk8s registry ---"
IMPORTED_OPERATOR_IMAGE_NAME="docker.io/library/${OPERATOR_IMAGE_BASE}:${OPERATOR_IMAGE_TAG}"
microk8s ctr image rm ${OPERATOR_IMAGE} || true
microk8s ctr image tag ${IMPORTED_OPERATOR_IMAGE_NAME} ${OPERATOR_IMAGE}
echo "Tagged ${IMPORTED_OPERATOR_IMAGE_NAME} as ${OPERATOR_IMAGE} in microk8s registry"

echo "--- Building agent_sdk wheel ---"
rm -f hello-agent/agent_sdk-*.whl
cd agent_sdk
python3 -m build
cd ..
cp agent_sdk/dist/agent_sdk-*.whl hello-agent/
echo "agent_sdk wheel built and copied to hello-agent/"

echo "--- Building Hello Agent Image with agentctl ---"
cd agentctl
go build -o agentctl .
./agentctl build --agent-name=hello-agent --image-tag=${HELLO_AGENT_IMAGE_TAG} --import-microk8s
cd ..
echo "Hello Agent Image Built and Imported: ${HELLO_AGENT_IMAGE}"

echo "--- Installing CRDs ---"
./scripts/install_crds.sh

echo "--- Deploying components using agentctl (Helm) ---"
cd agentctl
./agentctl deploy --agent-name=hello-agent --namespace=agent-hello-agent --image-tag=${HELLO_AGENT_IMAGE_TAG} \
  --set globalSecrets.enabled=false \
  --set postgresql.enabled=false \
  --set valkey.enabled=false \
  --set agentOperator.enabled=false \
  --set agent.name=hello-agent \
  --set agent.type=hello-agent \
  --set agent.image=${HELLO_AGENT_IMAGE}
cd ..

echo "--- Waiting for Hello Agent Pod to be Ready ---"
sleep 10
AGENT_POD_LABEL="agent-name=hello-agent"
AGENT_NAMESPACE="agent-hello-agent"

TIMEOUT=180
INTERVAL=5
ELAPSED=0
while true; do
  PHASE=$(kubectl get pod -n ${AGENT_NAMESPACE} -l ${AGENT_POD_LABEL} -o jsonpath='{.items[0].status.phase}' 2>/dev/null)
  READY=$(kubectl get pod -n ${AGENT_NAMESPACE} -l ${AGENT_POD_LABEL} -o jsonpath='{.items[0].status.conditions[?(@.type=="Ready")].status}' 2>/dev/null)
  if [ "$PHASE" == "Running" ] && [ "$READY" == "True" ]; then
    echo "Hello Agent Pod is Running and Ready."
    break
  fi
  if [ $ELAPSED -ge $TIMEOUT ]; then
    echo "Error: Timeout waiting for agent pod to be Running and Ready."
    kubectl get pods -n ${AGENT_NAMESPACE}
    exit 1
  fi
  sleep $INTERVAL
  ELAPSED=$((ELAPSED + INTERVAL))
  echo "Waiting... (Phase: ${PHASE:-'Unknown'}, Ready: ${READY:-'Unknown'}, ${ELAPSED}s / ${TIMEOUT}s)"
done

echo "--- Verification ---"
echo "Postgres Pod Status:"
kubectl get pods -l app.kubernetes.io/component=postgresql -n ${NAMESPACE}
echo ""
echo "Valkey Pod Status:"
kubectl get pods -l app.kubernetes.io/component=valkey -n ${NAMESPACE}
echo ""
echo "Operator Pod Status:"
kubectl get pods -l app.kubernetes.io/component=agent-operator -n ${NAMESPACE}
echo ""
echo "Agent Pod Status:"
kubectl get pods -l ${AGENT_POD_LABEL} -n ${AGENT_NAMESPACE}
echo ""
echo "Agent Custom Resource Status:"
kubectl get agent -n ${AGENT_NAMESPACE} -o yaml
echo ""
echo "Operator Logs:"
OPERATOR_POD=$(kubectl get pod -l app.kubernetes.io/component=agent-operator -n ${NAMESPACE} -o jsonpath='{.items[0].metadata.name}')
kubectl logs ${OPERATOR_POD} -n ${NAMESPACE} --tail=50
echo ""
echo "Agent Logs (via agentctl):"
cd agentctl
./agentctl logs hello-agent --namespace=agent-hello-agent
cd ..

echo "--- Testing agentctl status ---"
cd agentctl
./agentctl status
cd ..

echo "--- Deployment, Verification, and Testing Complete ---"
