Open Match Example Helm Chart
=============================

This chart installs the Open Match example clientloadgen, frontendclient, backendclient, and example MMF and evaluator.

To deploy this chart run:

```bash
helm upgrade --install --wait --debug open-match-example
    install/helm/open-match-example \
    --namespace=open-match \
    --set openmatch.image.registry=$(REGISTRY) \
    --set openmatch.image.tag=$(TAG)
```
