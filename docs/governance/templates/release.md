# v{version}

This is the {version} release of Open Match.

Check the [README](https://github.com/GoogleCloudPlatform/open-match/tree/release-{version}) for details on features, installation and usage.

Release Notes
-------------

{ insert enhancements from the changelog and/or security and breaking changes }

Images
------

See [CHANGELOG](https://github.com/GoogleCloudPlatform/open-match/blob/release-{version}/CHANGELOG.md) for more details on changes.

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
kubectl apply -f https://github.com/GoogleCloudPlatform/open-match/releases/download/v{version}/install.yaml --namespace open-match
# Install the example MMF and Evaluator.
kubectl apply -f https://github.com/GoogleCloudPlatform/open-match/releases/download/v{version}/install-example.yaml --namespace open-match
```
