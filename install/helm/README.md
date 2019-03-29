Open Match Helm Chart
=====================

Open Match provides a Helm chart to quickly 

```bash
# Install Helm and Tiller
# See https://github.com/helm/helm/releases for 
cd /tmp && curl -Lo helm.tar.gz https://storage.googleapis.com/kubernetes-helm/helm-v2.13.0-linux-amd64.tar.gz && tar xvzf helm.tar.gz --strip-components 1 && mv helm $(PREFIX)/bin/helm && mv tiller $(PREFIX)/bin/tiller

# Install Helm to Kubernetes Cluster
kubectl create serviceaccount --namespace kube-system tiller
helm init --service-account tiller --force-upgrade
kubectl create clusterrolebinding tiller-cluster-rule --clusterrole=cluster-admin --serviceaccount=kube-system:tiller
# Run if RBAC is enabled.
kubectl patch deploy --namespace kube-system tiller-deploy -p '{"spec":{"template":{"spec":{"serviceAccount":"tiller"}}}}'  

# Deploy Open Match
helm upgrade --install --wait --debug open-match \
    install/helm/open-match \
    --namespace=open-match \
    --set openmatch.image.registry=$(REGISTRY) \
    --set openmatch.image.tag=$(TAG)
```
