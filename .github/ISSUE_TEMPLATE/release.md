---
name: Publish a Release
about: Instructions and checklist for creating a release.
title: 'Release X.Y.Z-rc.N'
labels: kind/release
assignees: ''
---

# Open Match Release Process

Follow these instructions to create an Open Match release. The output of the
release process is new images and new configuration.

## Getting setup

**NOTE: The instructions below are NOT strictly copy-pastable and assume 0.5**
**release. Please update the version number for your commands.**

The Git flow for pushing a new release is similar to the development process
but there are some small differences.

### 1. Clone Repository

```shell
# Clone your fork of the Open Match repository.
git clone git@github.com:afeddersen/open-match.git
# Change directory to the git repository.
cd open-match
# Add a remote, you'll be pushing to this.
git remote add upstream https://github.com/googleforgames/open-match.git
```

### 2. Release Branch

If you're creating the first release of the version, that would be `0.5.0-rc.1`
then you'll need to create the release branch.

```shell
# Create a local release branch.
git checkout -b release-0.5 upstream/main
# Push the branch upstream.
git push upstream release-0.5
```

otherwise there should already be a `release-0.5` branch so run,

```shell
# Checkout the release branch.
git checkout -b release-0.5 upstream/release-0.5
```

**NOTE: The branch name must be in the format, `release-X.Y.Z` otherwise**
**some artifacts will not be pushed.**

## Releases & Versions

Open Match uses Semantic Versioning 2.0.0. If you're not familiar please
see the documentation - [https://semver.org/](https://semver.org/).

Full Release / Stable Release:

* The final software product. Stable, reliable, etc...
* Example: 1.0.0, 1.1.0

Release Candidate (RC):

* A release candidate (RC) is a version with the potential to be the final
  product but it hasn't validated by automated and/or manual tests.
* Example: 1.0.0-rc.1

Hot Fixes:

* Code developed to correct a major software bug or fault
  that's been discovered after the full release.
* Example: 1.0.1

Preview:

* Rare, a one off release cut from the main branch to provide early access
  to APIs or some other major change.
* **NOTE: There's no branch for this release.** 
* Example: 0.5-preview.1

**NOTE: Semantic versioning is enforced by `go mod`. A non-compliant version**
**tag will cause `go get` to break for users.**

# Detailed Instructions

## Find and replace

Below this point you will see {version} used as a placeholder for future
releases. Find {version} and replace with the current release (e.g. 0.5.0)

## Create a release branch in the upstream open-match repository

**Note: This step is performed by the person who starts the release.  It is
only required once.**

- [ ] Create the branch in the **upstream** repository. It should be named
  release-X.Y.Z. Example: release-0.5. At this point there's effectively a code
  freeze for this version and all work on main will be included in a future
  version. If you're on the branch that you created in the *getting setup*
  section above you should be able to push upstream.

```shell
git push origin release-0.5
```

- [ ] Announce a PR freeze on release-X.Y.Z branch on [open-match-discuss@](https://groups.google.com/forum/#!forum/open-match-discuss).
- [ ] Open the [`Makefile`](makefile-version) and change BASE_VERSION entry.
- [ ] Open the [`install/helm/open-match/Chart.yaml`](om-chart-yaml-version) and change the `appVersion` and `version` entries.
- [ ] Open the [`install/helm/open-match/values.yaml`](om-values-yaml-version) and change the `tag` entries.
- [ ] Open the [`cloudbuild.yaml`] and change the `_OM_VERSION` entry.
- [ ] There might be additional references to the old version but be careful not to change it for places that have it for historical purposes.
- [ ] Update usage requirements in the Installation doc - e.g. supported minikube version, kubectl version, golang version, etc.
- [ ] Create a PR with the changes, include the release candidate name, and point it to the release branch.
- [ ] Go to [open-match-build](https://pantheon.corp.google.com/cloud-build/triggers?project=open-match-build) and update all *post submit* triggers' `_GCB_LATEST_VERSION` value to the `X.Y.Z` of the release. This value should only increase as it's used to determine the latest stable version.
- [ ] Merge your changes once the PR is approved. Note: the helm chart is not published to the public registry until the merge is complete (it's a second cloud build trigger upon merge), so you won't be able to do final release testing until after all checks/approvals are finished!

## Create a release branch in the upstream open-match-docs repository
- [ ] Open [`Makefile`](makefile-version) and change BASE_VERSION entry.
- [ ] Open [`cloudbuild.yaml`] and change the `_OM_VERSION` entry.
- [ ] Open [`site/config.toml`] and change the `release_version` entry.
- [ ] Open [`site/static/swaggerui/config.json`] and change the `api/VERSION/...` entries
- [ ] Create a PR with the changes, include the release candidate name, and point it to the release branch.

## Complete Milestone

**Note: This step is performed by the person who starts the release. It is
only required once.**
- [ ] Create the next [version milestone](https://github.com/googleforgames/open-match/milestones) and use [semantic versioning](https://semver.org/) when naming it to be consistent with the [Go community](https://blog.golang.org/versioning-proposal).
- [ ] Create a *draft* [release](https://github.com/googleforgames/open-match/releases).  Note that github has both "Pre-release" and "draft" as different concepts for a release.  Until the release is finalized, only use "Save draft", and do not use "Publish release".
- [ ] Use the [release template](https://github.com/googleforgames/open-match/blob/main/docs/governance/templates/release.md)
  - [ ] `Tag = v{version}` (Example: v0.5.0. Append -rc.# for release candidates. Example: v0.5.0-rc.1.)
  - [ ] `Target = release-X.Y.Z` (Example: release-0.5.)
  - [ ] `Release Title = v{version}` (Must match `Tag`)
  - [ ] `Write` section will contain the contents from the [release template](https://github.com/googleforgames/open-match/blob/main/docs/governance/templates/release.md).
- [ ] Add the milestone to all PRs and issues that were merged since the last milestone. Look at the [releases page](https://github.com/googleforgames/open-match/releases) and look for the "X commits to main since this release" for the diff.
- [ ] Review all [milestone-less closed PRs](https://github.com/googleforgames/open-match/pulls?q=is%3Apr+is%3Aclosed+no%3Amilestone) and assign the appropriate milestone.
- [ ] Review all [PRs in milestone](https://github.com/googleforgames/open-match/milestones) for proper [labels](https://github.com/googleforgames/open-match/labels) (ex: area/build).
- [ ] View all open entries in milestone and move them to a future milestone if they aren't getting closed in time. https://github.com/googleforgames/open-match/milestones/v{version}
- [ ] Review all closed PRs against the milestone. Put the user visible changes into the release notes using the suggested format. https://github.com/googleforgames/open-match/pulls?utf8=%E2%9C%93&q=is%3Apr+is%3Aclosed+is%3Amerged+milestone%3Av{version}
- [ ] Review all closed issues against the milestone. Put the user visible changes into the release notes using the suggested format. https://github.com/googleforgames/open-match/issues?utf8=%E2%9C%93&q=is%3Aissue+is%3Aclosed+milestone%3Av{version}
- [ ] Verify everything in the [milestone](https://github.com/googleforgames/open-match/milestones) is complete with the exception of the release issue itself.

## Build And Test Artifacts

- [ ] Navigate to the [Cloud Console](https://console.cloud.google.com) in a browser and open the [Cloud Build History section](https://console.cloud.google.com/cloud-build/builds?project=open-match-build) and find the latest "Post Submit" build (trigger id: 9a451c7a-197b-4a38-a612-21f4c53c42fd) of the merged commit. The build may still be running, if so wait for it to finish. If it failed, fix the error and repeat this section. Open the build details and click on step 12, "Build: Docker Images". Take note of the docker image version tag near the top of the build log. This is the "{source version tag}" referenced in various commands below. Example: `0.5.0-a4706cb`.
- [ ] Run `./docs/governance/templates/release.sh {source version tag} {version}` to copy the images to open-match-public-images.
- [ ] If this is not a release candidate or preview but a full release, run `./docs/governance/templates/release.sh {source version tag} latest` to tag these public images as the default version to pull from the registry.
- [ ] Once the images have successfully been pushed to the registry, modify the line `open-match.dev/open-match v0.0.0-dev` in all `go.mod` files in the [Tutorials] (https://github.com/googleforgames/open-match/tree/main/tutorials) directory to use the current release version for the remainder of your local release testing. This includes all solution subdirectories as well. This change is local only and doesn't get committed to git.
- [ ] Copy the installation files named `{sequence_number}-{component}.yaml` (example: `01-open-match-core.yaml`) from the [build folder in the private open-match-build-artifacts GCS bucket https://storage.mtls.cloud.google.com/open-match-build-artifacts/{version}](https://console.cloud.google.com/storage/browser/open-match-build-artifacts?project=open-match-build) to the release draft you created. Download them to your local machine, and then attach them to the draft using the Github UI. Note: the `05-jaeger.yaml` file no longer exists after release 1.8, so don't be surprised if that number is missing.
- [ ] Update the [Slack invitation link](https://slack.com/help/articles/201330256-invite-new-members-to-your-workspace#share-an-invite-link) in [open-match.dev](https://open-match.dev/site/docs/contribute/#get-involved).
- [ ] Test Open Match installation under GKE and Minikube enviroment using the YAML files attached to the release and the latest Helm chart, pulled from the public helm repo (not your local copy from github). Follow the [First Match](https://development.open-match.dev/site/docs/getting-started/first_match/) guide, run `make proxy-demo`, and open `localhost:51507` to make sure everything works.
  - [ ] Minikube: Run `make create-mini-cluster` to create a local cluster with latest Kubernetes API version.
  - [ ] GKE: Run `make create-gke-cluster` to create a GKE cluster.
  - [ ] Helm: Run `helm install open-match -n open-match open-match/open-match`. Note, the helm chart for the release is not public until the PR has been merged, so you cannot complete this step until after the PR is closed and the 'Tagged Build' trigger (trigger ID: 083adc1a-fcac-4033-bc38-b9f6eadcb75d) has completed, which publishes the helm chart.

## Finalize

- [ ] Make sure your release draft reflects all steps up to this point, and is saved (so contributors can review it).
- [ ] Circulate the draft release to active contributors.  Where reasonable, get everyone's ok on the release notes before continuing.
- [ ] Publish the [Release](om-release) in Github.  This will notify repository watchers.
- [ ] Publish the [Release](om-release) on Open Match [Blog](https://open-match.dev/site/blog/).

## Announce

- [ ] Send an email to the [mailing list](https://groups.google.com/forum/#!newtopic/open-match-discuss) with the release details (copy-paste the release blog post)
- [ ] Send a chat on the [Slack channel](https://open-match.slack.com/). "Open Match {version} has been released! Check it out at {release url}."

[makefile-version]: https://github.com/googleforgames/open-match/blob/main/Makefile#L53
[om-chart-yaml-version]: https://github.com/googleforgames/open-match/blob/main/install/helm/open-match/Chart.yaml#L16
[om-values-yaml-version]: https://github.com/googleforgames/open-match/blob/main/install/helm/open-match/values.yaml#L16
[om-release]: https://github.com/googleforgames/open-match/releases/new
[readme-deploy]: https://github.com/googleforgames/open-match/blob/main/README.md#deploy-to-kubernetes
