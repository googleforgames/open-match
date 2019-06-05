# v{version}

This is the {version} release of Open Match.

Check the [README](https://github.com/googleforgames/open-match/tree/release-{version}) for details on features, installation and usage.

Release Notes
-------------

{ insert enhancements from the changelog and/or security and breaking changes }

**Breaking Changes**
 * API Changed #PR

**Enhancements**
 * New Harness #PR

**Security Fixes**
 * Reduced privileges required for MMF. #PR

See [CHANGELOG](https://github.com/googleforgames/open-match/blob/release-{version}/CHANGELOG.md) for more details on changes.

Images
------

```bash
# Servers
docker pull gcr.io/open-match-public-images/openmatch-backendapi:{version}
docker pull gcr.io/open-match-public-images/openmatch-frontendapi:{version}
docker pull gcr.io/open-match-public-images/openmatch-mmforc:{version}
docker pull gcr.io/open-match-public-images/openmatch-mmlogicapi:{version}

# Evaluators
docker pull gcr.io/open-match-public-images/openmatch-evaluator-serving:{version}

# Sample Match Making Functions
docker pull gcr.io/open-match-public-images/openmatch-mmf-go-simple:{version}

# Test Clients
docker pull gcr.io/open-match-public-images/openmatch-backendclient:{version}
docker pull gcr.io/open-match-public-images/openmatch-clientloadgen:{version}
docker pull gcr.io/open-match-public-images/openmatch-frontendclient:{version}
```

_This software is currently alpha, and subject to change. Not to be used in production systems._

Installation
------------

To deploy Open Match in your Kubernetes cluster run the following commands:

```bash
# Grant yourself cluster-admin permissions so that you can deploy service accounts.
kubectl create clusterrolebinding myname-cluster-admin-binding --clusterrole=cluster-admin --user=$(YOUR_KUBERNETES_USER_NAME)
# Place all Open Match components in their own namespace.
kubectl create namespace open-match
# Install Open Match and monitoring services.
kubectl apply -f https://github.com/googleforgames/open-match/releases/download/v{version}/install.yaml --namespace open-match
# Install the demo.
kubectl apply -f https://github.com/googleforgames/open-match/releases/download/v{version}/install-demo.yaml --namespace open-match
```
