#!/bin/sh
kubectl get $(kubectl get fleetallocation -o name) -o jsonpath='{.status.gameServer.status.address}{":"}{.status.gameServer.status.ports[0].port}'
