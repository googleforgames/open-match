# install/yaml
This directory contains Kubernetes YAML resource definitions, which should be applied according to their filename order.  Only Redis & Open Match are required, Prometheus is optional.
```
kubectl apply -f 01-redis.yaml
kubectl apply -f 02-open-match.yaml
```
**Note**: Trying to apply the Kubernetes Prometheus Operator resource definition files without a cluster-admin rolebinding on GKE doesn't work without running the following command first. See https://github.com/coreos/prometheus-operator/issues/357
```
kubectl create clusterrolebinding projectowner-cluster-admin-binding --clusterrole=cluster-admin --user=<GCP_ACCOUNT>
```
```
kubectl apply -f 03-prometheus.yaml
```
[There is a known dependency ordering issue when applying the Prometheus resource; just wait a couple moments and apply it again.](https://github.com/GoogleCloudPlatform/open-match/issues/46)

[Accurate as of v0.2.0] Output from `kubectl get all` if everything succeeded should look something like this:
```
NAME                                       READY     STATUS    RESTARTS   AGE
pod/om-backendapi-84bc9d8fff-q89kr         1/1       Running   0          9m
pod/om-frontendapi-55d5bb7946-c5ccb        1/1       Running   0          9m
pod/om-mmforc-85bfd7f4f6-wmwhc             1/1       Running   0          9m
pod/om-mmlogicapi-6488bc7fc6-g74dm         1/1       Running   0          9m
pod/prometheus-operator-5c8774cdd8-7c5qm   1/1       Running   0          9m
pod/prometheus-prometheus-0                2/2       Running   0          9m
pod/redis-master-9b6b86c46-b7ggn           1/1       Running   0          9m

NAME                          TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)          AGE
service/kubernetes            ClusterIP   10.59.240.1     <none>        443/TCP          19m
service/om-backend-metrics    ClusterIP   10.59.254.43    <none>        29555/TCP        9m
service/om-backendapi         ClusterIP   10.59.240.211   <none>        50505/TCP        9m
service/om-frontend-metrics   ClusterIP   10.59.246.228   <none>        19555/TCP        9m
service/om-frontendapi        ClusterIP   10.59.250.59    <none>        50504/TCP        9m
service/om-mmforc-metrics     ClusterIP   10.59.240.59    <none>        39555/TCP        9m
service/om-mmlogicapi         ClusterIP   10.59.248.3     <none>        50503/TCP        9m
service/prometheus            NodePort    10.59.252.212   <none>        9090:30900/TCP   9m
service/prometheus-operated   ClusterIP   None            <none>        9090/TCP         9m
service/redis                 ClusterIP   10.59.249.197   <none>        6379/TCP         9m

NAME                                        DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
deployment.extensions/om-backendapi         1         1         1            1           9m
deployment.extensions/om-frontendapi        1         1         1            1           9m
deployment.extensions/om-mmforc             1         1         1            1           9m
deployment.extensions/om-mmlogicapi         1         1         1            1           9m
deployment.extensions/prometheus-operator   1         1         1            1           9m
deployment.extensions/redis-master          1         1         1            1           9m

NAME                                                   DESIRED   CURRENT   READY     AGE
replicaset.extensions/om-backendapi-84bc9d8fff         1         1         1         9m
replicaset.extensions/om-frontendapi-55d5bb7946        1         1         1         9m
replicaset.extensions/om-mmforc-85bfd7f4f6             1         1         1         9m
replicaset.extensions/om-mmlogicapi-6488bc7fc6         1         1         1         9m
replicaset.extensions/prometheus-operator-5c8774cdd8   1         1         1         9m
replicaset.extensions/redis-master-9b6b86c46           1         1         1         9m

NAME                                  DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/om-backendapi         1         1         1            1           9m
deployment.apps/om-frontendapi        1         1         1            1           9m
deployment.apps/om-mmforc             1         1         1            1           9m
deployment.apps/om-mmlogicapi         1         1         1            1           9m
deployment.apps/prometheus-operator   1         1         1            1           9m
deployment.apps/redis-master          1         1         1            1           9m

NAME                                             DESIRED   CURRENT   READY     AGE
replicaset.apps/om-backendapi-84bc9d8fff         1         1         1         9m
replicaset.apps/om-frontendapi-55d5bb7946        1         1         1         9m
replicaset.apps/om-mmforc-85bfd7f4f6             1         1         1         9m
replicaset.apps/om-mmlogicapi-6488bc7fc6         1         1         1         9m
replicaset.apps/prometheus-operator-5c8774cdd8   1         1         1         9m
replicaset.apps/redis-master-9b6b86c46           1         1         1         9m

NAME                                     DESIRED   CURRENT   AGE
statefulset.apps/prometheus-prometheus   1         1         9m
```
