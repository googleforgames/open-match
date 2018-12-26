#!/bin/bash
kubectl apply -f 01-fleet.yaml
UNSTARTED=true
while $UNSTARTED; do
    kubectl get pods | grep udp-server | grep Running 2>&1 1>/dev/null
    if [ ${?} -eq 0 ]; then
        UNSTARTED=false 
    fi
done
echo "Game server started."
kubectl create -f 02-fleetallocation.yaml
CONNSTRING=$(bash 03-get-gameserver-addr.sh)
echo "Server available, asking for match to assign to ${CONNSTRING}"
kubectl run --rm --restart=Never --image-pull-policy=Always -i --tty \
    --image=gcr.io/matchmaker-dev-201405/openmatch-backendclient:dev mm \
    --command ./backendclient -- "${CONNSTRING}"
