# !/usr/bin/env bash
KNATIVE_NET=${KNATIVE_NET:-kourier}
KNATIVE_VERSION=${KNATIVE_VERSION:-1.2.2}
KNATIVE_EVENTING_VERSION=${KNATIVE_EVENTING_VERSION:-1.2.0}
NAMESPACE=${NAMESPACE:-default}
BROKER_NAME=${BROKER_NAME:-default}
KNATIVE_NET_KOURIER_VERSION=${KNATIVE_NET_KOURIER_VERSION:-1.2.0}
TRIGGERMESH_VERSION=${TRIGGERMESH_VERSION:-v1.20.0}

set -eo pipefail
set -u

echo -e "Switching to docker-desktop kubectl context"
kubectl config use-context docker-desktop

STARTTIME=$(date +%s)
echo -e "🍿 Installing Knative Serving... \033[0m"

n=0
set +e
until [ $n -ge 2 ]; do
  kubectl apply -f https://github.com/knative/serving/releases/download/knative-v${KNATIVE_VERSION}/serving-crds.yaml > /dev/null && break
  echo "Serving CRDs failed to install on first try"
  n=$[$n+1]
  sleep 5
done
set -e
kubectl wait --for=condition=Established --all crd > /dev/null

n=0
set +e
until [ $n -ge 2 ]; do
  kubectl apply -f https://github.com/knative/serving/releases/download/knative-v${KNATIVE_VERSION}/serving-core.yaml > /dev/null && break
  echo "Serving Core failed to install on first try"
  n=$[$n+1]
  sleep 5
done
set -e
kubectl wait pod --timeout=-1s --for=condition=Ready -l '!job-name' -n knative-serving > /dev/null

echo -e "🔌 Installing Knative Serving Networking Layer ${KNATIVE_NET}... \033[0m"

## INSTALL KOURIER
n=0
until [ $n -ge 3 ]; do
  kubectl apply -f https://github.com/knative-sandbox/net-kourier/releases/download/knative-v${KNATIVE_NET_KOURIER_VERSION}/kourier.yaml > /dev/null && break
  echo "Kourier failed to install on first try"
  n=$[$n+1]
  sleep 10
done
kubectl wait pod --timeout=-1s --for=condition=Ready -l '!job-name' -n kourier-system > /dev/null
kubectl wait pod --timeout=-1s --for=condition=Ready -l '!job-name' -n knative-serving > /dev/null

# Configure Knative to use this ingress
kubectl patch configmap/config-network \
  --namespace knative-serving \
  --type merge \
  --patch '{"data":{"ingress.class":"kourier.ingress.networking.knative.dev"}}'

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: kourier-ingress
  namespace: kourier-system
  labels:
    networking.knative.dev/ingress-provider: kourier
spec:
  type: NodePort
  selector:
    app: 3scale-kourier-gateway
  ports:
    - name: http2
      nodePort: 31080
      port: 80
      targetPort: 8080
EOF

# Setup Knative DOMAIN DNS
INGRESS_HOST=$(kubectl -n kourier-system get service kourier -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
if [ -z $INGRESS_HOST ]; then INGRESS_HOST=$(kubectl -n kourier-system get service kourier -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'); fi
while [ -z $INGRESS_HOST ]; do
  sleep 5
  INGRESS_HOST=$(kubectl -n kourier-system get service kourier -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
  if [ -z $INGRESS_HOST ]; then INGRESS_HOST=$(kubectl -n kourier-system get service kourier -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'); fi
done

if [ "$INGRESS_HOST" == "localhost" ]; then INGRESS_HOST=127.0.0.1; fi

KNATIVE_DOMAIN=$INGRESS_HOST.sslip.io
kubectl patch configmap -n knative-serving config-domain -p "{\"data\": {\"$KNATIVE_DOMAIN\": \"\"}}"

echo -e "🔥 Installing Knative Eventing... \033[0m"
n=0
until [ $n -ge 2 ]; do
  kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v${KNATIVE_EVENTING_VERSION}/eventing-crds.yaml > /dev/null && break
  echo "Eventing CRDs failed to install on first try"
  n=$[$n+1]
  sleep 5
done
kubectl wait --for=condition=Established --all crd > /dev/null
n=0
until [ $n -ge 2 ]; do
  kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v${KNATIVE_EVENTING_VERSION}/eventing-core.yaml > /dev/null && break
  echo "Eventing Core failed to install on first try"
  n=$[$n+1]
  sleep 5
done
kubectl wait pod --timeout=-1s --for=condition=Ready -l '!job-name' -n knative-eventing > /dev/null
n=0
until [ $n -ge 2 ]; do
  kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v${KNATIVE_EVENTING_VERSION}/in-memory-channel.yaml > /dev/null && break
  echo "Eventing Memory Channel failed to install on first try"
  n=$[$n+1]
  sleep 5
done
kubectl wait pod --timeout=-1s --for=condition=Ready -l '!job-name' -n knative-eventing > /dev/null
n=0
until [ $n -ge 2 ]; do
  kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v${KNATIVE_EVENTING_VERSION}/mt-channel-broker.yaml > /dev/null && break
  echo "Eventing MT Memory Broker failed to install on first try"
  n=$[$n+1]
  sleep 5
done
kubectl wait pod --timeout=-1s --for=condition=Ready -l '!job-name' -n knative-eventing > /dev/null

kubectl apply -f - <<EOF
apiVersion: eventing.knative.dev/v1
kind: broker
metadata:
 name: ${BROKER_NAME}
 namespace: ${NAMESPACE}
EOF

echo -e "🦾 Installing Triggermesh ... \033[0m"
echo "Setting up Triggermesh"
# kubectl apply -f https://github.com/triggermesh/triggermesh/releases/download/${TRIGGERMESH_VERSION}/triggermesh-crds.yaml
# kubectl wait --for=condition=Established --all crd
# kubectl apply -f https://github.com/triggermesh/triggermesh/releases/download/${TRIGGERMESH_VERSION}/triggermesh.yaml
# kubectl wait pod --timeout=-1s --for=condition=Ready -l '!job-name' -n triggermesh
# kubectl -n ${NAMESPACE} get broker ${BROKER_NAME}
DURATION=$(($(date +%s) - $STARTTIME))
echo -e "\033[0;92m 🚀 Triggermesh prerequiste install took: $(($DURATION / 60))m$(($DURATION % 60))s \033[0m"
echo -e "\033[0;92m 🎉 Now have some fun developing Serverless and Event Driven Apps \033[0m"
