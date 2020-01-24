# v{version}

This is the {version} release of Open Match.

Check the [official website](https://open-match.dev) for details on features, installation and usage.

Release Notes
-------------

**Feature Highlights**
{ highlight here the most notable changes and themes at a high level}

**Breaking Changes**
{ detail any behaviors or API surfaces which worked in a previous version which will no longer work correctly }

> Future releases towards 1.0.0 may still have breaking changes.

**Security Fixes**
{ list any changes which fix vulnerabilities in open match }

**Enhancements**
{ go into details on improvements and changes }

Usage Requirements
-------------
* Tested against Kubernetes Version { a list of k8s versions}
* Golang Version = v{ required golang version }

Images
------

```bash
# Servers
docker pull gcr.io/open-match-public-images/openmatch-backend:{version}
docker pull gcr.io/open-match-public-images/openmatch-frontend:{version}
docker pull gcr.io/open-match-public-images/openmatch-query:{version}
docker pull gcr.io/open-match-public-images/openmatch-synchronizer:{version}

# Evaluators
docker pull gcr.io/open-match-public-images/openmatch-evaluator-go-simple:{version}

# Sample Match Making Functions
docker pull gcr.io/open-match-public-images/openmatch-mmf-go-soloduel:{version}
docker pull gcr.io/open-match-public-images/openmatch-mmf-go-pool:{version}

# Test Clients
docker pull gcr.io/open-match-public-images/openmatch-demo-first-match:{version}
```

_This software is currently alpha, and subject to change. Not to be used in production systems._

Installation
------------

* Follow [Open Match Installation Guide](https://open-match.dev/site/docs/installation/) to setup Open Match in your cluster.

API Definitions
------------

- gRPC API Definitions are available in [API references](https://open-match.dev/site/docs/reference/api/) - _Preferred_
- HTTP API Definitions are available in [SwaggerUI](https://open-match.dev/site/swaggerui/index.html)
