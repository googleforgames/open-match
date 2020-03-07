### Open Match Helm Chart Templates
This directory contains the [helm](https://helm.sh/ "helm") chart templates used to customize and deploy Open Match.

Templates under the `templates/` directory are for the core components in Open Match - e.g. backend, frontend, query, synchronizor, some security policies, and configmaps are defined under this folder.

Open Match also provides templates for optional components that are disabled by default under the `subcharts/` directory.
1. `open-match-customize` contains flexible templates to deploy your own matchfunction and evaluator.
2. `open-match-telemetry` contains monitoring supports for Open Match, you may choose to enable/disable [jaeger](https://www.jaegertracing.io/ "jaeger"), [prometheus](http://prometheus.io "prometheus"), [stackdriver](https://cloud.google.com/stackdriver/ "stackdriver"), and [grafana](https://grafana.com/ "grafana") by overriding the config values in the provided templates.

You may control the behavior of Open Match by overriding the configs in `install/helm/open-match/values.yaml` file. Here are a few examples:

```diff
# install/helm/open-match/values.yaml
# 1. Configs under the `global` section affects all components - including components in the subcharts.
# 2. Configs under the subchart name - e.g. `open-match-customize` only affects the settings in that subchart.
# 3. Otherwise, the configs are for core components (templates in the parent chart) only.

# Overrides spec.type of a specific Kubernetes Service
# Equivalent helm cli flag --set swaggerui.portType=LoadBalancer
swaggerui:
-  portType: ClusterIP
+  portType: LoadBalancer

# Overrides spec.type of all Open Match components - including components in the subcharts
# Equivalent helm cli flag --set global.kubernetes.service.portType=LoadBalancer
global:
  kubernetes:
    service:
-	  portType: ClusterIP
+     portType: LoadBalancer

# Enables grafana support in Open Match
# Equivalent helm cli flag --set global.telemetry.grafana.enabled=true
global:
  telemetry:
    grafana:
-     enabled: false
+     enabled: true

# Enables an optional component in Open Match
# Equivalent helm cli flag --set open-match-telemetry.enabled=true
open-match-telemetry:
- enabled: false
+ enabled: true

# Enables rpc logging in Open Match
# Equivalent helm cli flag --set global.logging.rpc.enabled=true
global:
  logging:
    rpc:
-     enabled: false
+     enabled: true

# Instructs Open Match to use customized matchfunction and evaluator images
# Equivalent helm cli flag --set open-match-customize.image.registry=[XXX],open-match-customize.image.tag=[XXX]
open-match-customize:
  enabled: true
+   image:
+     registry: [YOUR_REGISTRY_URL]
+     tag: [YOUR_IMAGE_TAG]
+   function:
+     image: [YOUR_MATCHFUNCTION_IMAGE_NAME]
+   evaluator:
+     image: [YOUR_EVALUATOR_IMAGE_NAME]
```

Please see [Helm - Chart Template Guide](https://helm.sh/docs/chart_template_guide/#the-chart-template-developer-s-guide "Chart Template Guide") for the advanced usages and our [Makefile](https://github.com/googleforgames/open-match/blob/master/Makefile#L358 "Makefile")  for how we use the helm charts to deploy Open Match.
