# Install Open Match using Helm

This chart installs the Open Match application and defines deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

To deploy this chart run:

```bash
helm upgrade --install --wait --debug open-match install/helm/open-match \
    --namespace=open-match \
    --set openmatch.image.registry=$(REGISTRY) \
    --set openmatch.image.tag=$(TAG)
```
