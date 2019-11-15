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
git checkout -b release-0.5 upstream/master
# Push the branch upstream.
git push upstream release-0.5
```

otherwise there should already be a `release-0.5` branch so run,

```shell
# Checkout the release branch.
git checkout -b release-0.5 upstream/release-0.5
```

**NOTE: The branch name must be in the format, `release-X.Y` otherwise**
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

* Rare, a one off release cut from the master branch to provide early access
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
  release-X.Y. Example: release-0.5. At this point there's effectively a code
  freeze for this version and all work on master will be included in a future
  version. If you're on the branch that you created in the *getting setup*
  section above you should be able to push upstream.

```shell
git push origin release-0.5
```

- [ ] Announce a PR freeze on release-X.Y branch on [open-match-discuss@](mailing-list-post).
- [ ] Open the [`Makefile`](makefile-version) and change BASE_VERSION entry.
- [ ] Open the [`install/helm/open-match/Chart.yaml`](om-chart-yaml-version) and change the `appVersion` and `version` entries.
- [ ] Open the [`install/helm/open-match/values.yaml`](om-values-yaml-version) and change the `tag` entries.
- [ ] Open the [`cloudbuild.yaml`] and change the `_OM_VERSION` entry.
- [ ] There might be additional references to the old version but be careful not to change it for places that have it for historical purposes.
- [ ] Run `make release`
- [ ] Run `make api/api.md` in open-match repo to update the auto-generated API references in open-match-docs repo.
- [ ] Use the files under the `build/release/` directory for the Open Match installation guide. Make sure the artifacts work as expected - these are the artifacts that will be published to the GCS bucket and used in our release assets.
- [ ] Create a PR with the changes, include the release candidate name, and point it to the release branch.
- [ ] Go to [open-match-build](https://pantheon.corp.google.com/cloud-build/triggers?project=open-match-build) and update all *post submit* triggers' `_GCB_LATEST_VERSION` value to the `X.Y` of the release. This value should only increase as it's used to determine the latest stable version.
- [ ] Merge your changes once the PR is approved.

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
- [ ] Use the [release template](https://github.com/googleforgames/open-match/blob/master/docs/governance/templates/release.md)
  - [ ] `Tag` = v{version}. Example: v0.5.0. Append -rc.# for release candidates. Example: v0.5.0-rc.1.
  - [ ] `Target` = release-X.Y. Example: release-0.5.
  - [ ] `Release Title` = `Tag`
  - [ ] `Write` section will contain the contents from the [release template](https://github.com/googleforgames/open-match/blob/master/docs/governance/templates/release.md).
- [ ] Add the milestone to all PRs and issues that were merged since the last milestone. Look at the [releases page](https://github.com/googleforgames/open-match/releases) and look for the "X commits to master since this release" for the diff.
- [ ] Review all [milestone-less closed issues](https://github.com/googleforgames/open-match/issues?q=is%3Aissue+is%3Aclosed+no%3Amilestone) and assign the appropriate milestone.
- [ ] Review all [issues in milestone](https://github.com/googleforgames/open-match/milestones) for proper [labels](https://github.com/googleforgames/open-match/labels) (ex: area/build).
- [ ] Review all [milestone-less closed PRs](https://github.com/googleforgames/open-match/pulls?q=is%3Apr+is%3Aclosed+no%3Amilestone) and assign the appropriate milestone.
- [ ] Review all [PRs in milestone](https://github.com/googleforgames/open-match/milestones) for proper [labels](https://github.com/googleforgames/open-match/labels) (ex: area/build).
- [ ] View all open entries in milestone and move them to a future milestone if they aren't getting closed in time. https://github.com/googleforgames/open-match/milestones/v{version}
- [ ] Review all closed PRs against the milestone. Put the user visible changes into the release notes using the suggested format. https://github.com/googleforgames/open-match/pulls?utf8=%E2%9C%93&q=is%3Apr+is%3Aclosed+is%3Amerged+milestone%3Av{version}
- [ ] Review all closed issues against the milestone. Put the user visible changes into the release notes using the suggested format. https://github.com/googleforgames/open-match/issues?utf8=%E2%9C%93&q=is%3Aissue+is%3Aclosed+milestone%3Av{version}
- [ ] Verify the [milestone](https://github.com/googleforgames/open-match/milestones) is effectively 100% at this point with the exception of the release issue itself.

## Build Artifacts

- [ ] Go to the History section and find the "Post Submit" build of the merged commit that's running. Wait for it to go Green. If it's red, fix error repeat this section. Take note of the docker image version tag for next step. Example: 0.5.0-a4706cb.
- [ ] Run `./docs/governance/templates/release.sh {source version tag} {version}` to copy the images to open-match-public-images.
- [ ] If this is a new minor version in the newest major version then run `./docs/governance/templates/release.sh {source version tag} latest`.
- [ ] Copy the files from `build/release/` generated from `make release` to the release draft you created.  You can drag and drop the files using the Github UI.
- [ ] Update [Slack invitation link](https://slack.com/help/articles/201330256-invite-new-members-to-your-workspace#share-an-invite-link) in [open-match.dev](https://open-match.dev/site/docs/contribute/#get-involved).
- [ ] Test Open Match installation under GKE and Minikube enviroment using YAML files and Helm. Follow the [First Match](https://development.open-match.dev/site/docs/getting-started/first_match/) guide, run `make proxy-demo`, and open `localhost:51507` to make sure everything works.
  - [ ] Minikube: Run `make create-mini-cluster` to create a local cluster with latest Kubernetes API version.
  - [ ] GKE: Run `make create-gke-cluster` to create a GKE cluster.
  - [ ] Helm: Run `helm install open-match -n open-match open-match/open-match`
- [ ] Update usage requirements in the Installation doc - e.g. supported minikube version, kubectl version, golang version, etc.

## Finalize

- [ ] Save the release as a draft.
- [ ] Circulate the draft release to active contributors.  Where reasonable, get everyone's ok on the release notes before continuing.
- [ ] Publish the [Release](om-release) in Github.  This will notify repository watchers.

## Announce

- [ ] Send an email to the [mailing list](mailing-list-post) with the release details (copy-paste the release blog post)
- [ ] Send a chat on the [Slack channel](om-slack). "Open Match {version} has been released! Check it out at {release url}."

[om-slack]: https://open-match.slack.com/
[mailing-list-post]: https://groups.google.com/forum/#!newtopic/open-match-discuss
[release-template]: https://github.com/googleforgames/open-match/blob/master/docs/governance/templates/release.md
[makefile-version]: https://github.com/googleforgames/open-match/blob/master/Makefile#L53
[om-chart-yaml-version]: https://github.com/googleforgames/open-match/blob/master/install/helm/open-match/Chart.yaml#L16
[om-values-yaml-version]: https://github.com/googleforgames/open-match/blob/master/install/helm/open-match/values.yaml#L16
[om-release]: https://github.com/googleforgames/open-match/releases/new
[readme-deploy]: https://github.com/googleforgames/open-match/blob/master/README.md#deploy-to-kubernetes
