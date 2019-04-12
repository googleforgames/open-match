---
title: "Usage"
linkTitle: "Usage"
date: 2017-01-05
description: >
  A short lead descripton about this content page. It can be **bold** or _italic_ and can be split over multiple paragraphs.
---

## Usage

Documentation and usage guides on how to set up and customize Open Match.

### Precompiled container images

Once we reach a 1.0 release, we plan to produce publicly available (Linux) Docker container images of major releases in a public image registry. Until then, refer to the 'Compiling from source' section below.

### Compiling from source

All components of Open Match produce (Linux) Docker container images as artifacts, and there are included `Dockerfile`s for each. [Google Cloud Platform Cloud Build](https://cloud.google.com/cloud-build/docs/) users will also find `cloudbuild.yaml` files for each component in the corresponding `cmd/<COMPONENT>` directories.

All the core components for Open Match are written in Golang and use the [Dockerfile multistage builder pattern](https://docs.docker.com/develop/develop-images/multistage-build/). This pattern uses intermediate Docker containers as a Golang build environment while producing lightweight, minimized container images as final build artifacts. When the project is ready for production, we will modify the `Dockerfile`s to uncomment the last build stage. Although this pattern is great for production container images, it removes most of the utilities required to troubleshoot issues during development.

## Configuration
Currently, each component reads a local config file `matchmaker_config.json`, and all components assume they have the same configuration. To this end, there is a single centralized config file located in the `<REPO_ROOT>/config/` which is symlinked to each component's subdirectory for convenience when building locally. When `docker build`ing the component container images, the Dockerfile copies the centralized config file into the component directory.

We plan to replace this with a Kubernetes-managed config with dynamic reloading. 

### Guides
* Production guide - Lots of best practices to be written here before 1.0 release, right now it's a scattered collection of notes. **WIP**
* Development guide