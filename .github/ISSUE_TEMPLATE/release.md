---
name: release
about: Instructions and checklist for creating a release.
title: 'Release X.Y.Z-rc.N'
labels: kind/release
assignees: ''
---

# Open Match Release Process

Follow these instructions to create an Open Match release.  The output of the
release process is new images and new configuration.

## Getting setup

*note: the commands below are pasted from the 0.5 release.  make the necessary
changes to match your naming & environment.*

The Git flow for pushing a new release is similar to the development process
but there are some small differences.

**1. Clone your fork of the Open Match repository.**

```shell
git clone git@github.com:afeddersen/open-match.git
```
**2. Move into the new open-match directory.**

```shell
cd open-match
```

**3. Configure a remote that points to the upstream repository. This is required to sync changes you make in a fork with the original repository.  Note: Upstream is the gatekeeper of the project or the source of truth to which you wish to contribute.**

```shell
git remote add upstream https://github.com/googleforgames/open-match.git
```

**3. Fetch the branches and their respective commits from the upstream repo.**

```shell
git fetch upstream
```

**4.  Create a local release branch that tracks upstream and check it out.**

```shell
git checkout -b release-0.5 upstream/release-0.5
```

## Releases & Versions


Open Match uses Semantic Versioning 2.0.0.  If you're not familiar please
see the documentation - [https://semver.org/](https://semver.org/).

Full Release / Stable Release:

* The final software product.  Stable, reliable, etc...
* Naming example: 1.0.0

Release Candidate (RC):

* A release candidate (RC) is a version with the potential to be the final
  product but it hasn't validated by automated and/or manual tests.
* Naming example: 1.0.0-rc.1

Hot Fixes:

* Code developed to correct a major software bug or fault
  that's been discovered after the full release.
* Naming example: 1.0.1

# Detailed Instructions


## Find and replace


Below this point you will see {version} used as a placeholder for future
releases.  Find {version} and replace with the current release (e.g. 0.5.0)

## Create a release branch in the upstream repository


**Note: This step is performed by the person who starts the release.  It is
only required once.**

- [ ] Create the branch in the **upstream** repository. It should be named
  release-X.Y. Example: release-0.5. At this point there's effectively a code
  freeze for this version and all work on master will be included in a future
  version.  If you're on the branch that you created in the *getting setup*
  section above you should be able to push upstream.

```shell
git push origin release-0.5
```

- [ ] Announce a PR freeze on release-X.Y branch on [open-match-discuss@](mailing-list-post).
- [ ] Open the [`Makefile`](makefile-version) and change BASE_VERSION entry.
- [ ] Open the [`install/helm/open-match/Chart.yaml`](om-chart-yaml-version) and [`install/helm/open-match-example/Chart.yaml`](om-example-chart-yaml-version) and change the `appVersion` and `version` entries.
- [ ] Open the [`install/helm/open-match/values.yaml`](om-values-yaml-version) and [`install/helm/open-match-example/values.yaml`](om-example-values-yaml-version) and change the `tag` entries.
- [ ] Open the [`site/config.toml`] and change the `release_branch` and `release_version` entries.
- [ ] Open the [`cloudbuild.yaml`] and change the `_OM_VERSION` entry.
- [ ] Run `make clean release`
- [ ] There might be additional references to the old version but be careful not to change it for places that have it for historical purposes.
- [ ] Create a PR with the changes and include the release candidate name.
- [ ] Merge your changes once the PR is approved.

## Complete Milestone


**Note: This step is performed by the person who starts the release.  It is
only required once.**
- [ ] Create the next [version milestone](https://github.com/googleforgames/open-match/milestones) and use [semantic versioning](https://semver.org/) when naming it to be consistent with the [Go community](https://blog.golang.org/versioning-proposal).
- [ ] Create a *draft* [release](https://github.com/googleforgames/open-match/releases).
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

TODO: Add guidelines for labeling issues.

## Build Artifacts

- [ ] Go to [Cloud Build](https://pantheon.corp.google.com/cloud-build/triggers?project=open-match-build), under Post Submit click "Run Trigger".
- [ ] Go to the History section and find the "Post Submit" build that's running. Wait for it to go Green. If it's red, fix error repeat this section. Take note of the docker image version tag for next step. Example: 0.5.0-a4706cb.
- [ ] Run `./docs/governance/templates/release.sh {source version tag} {version}` to copy the images to open-match-public-images.
- [ ] If this is a new minor version in the newest major version then run `./docs/governance/templates/release.sh {source version tag} latest`.
- [ ] Copy the files from `build/release/` generated from `make release` to the release draft you created.  You can drag and drop the files using the Github UI.
- [ ] Run `make delete-gke-cluster create-gke-cluster` and run through the instructions under the [README](readme-deploy), verify the pods are healthy. You'll need to adjust the path to the `build/release/install.yaml` and `build/release/install-demo.yaml` in your local clone since you haven't published them yet.
- [ ] Open the [`README.md`](readme-deploy) update the version references and submit. (Release candidates can ignore this step.)
- [ ] Publish the [Release](om-release) in Github.

## Announce

- [ ] Send an email to the [mailing list](mailing-list-post) with the release details (copy-paste the release blog post)
- [ ] Send a chat on the [Slack channel](om-slack). "Open Match {version} has been released! Check it out at {release url}."

[om-slack]: https://open-match.slack.com/
[mailing-list-post]: https://groups.google.com/forum/#!newtopic/open-match-discuss
[release-template]: https://github.com/googleforgames/open-match/blob/master/docs/governance/templates/release.md
[makefile-version]: https://github.com/googleforgames/open-match/blob/master/Makefile#L53
[om-example-chart-yaml-version]: https://github.com/googleforgames/open-match/blob/master/install/helm/open-match/Chart.yaml#L16
[om-example-values-yaml-version]: https://github.com/googleforgames/open-match/blob/master/install/helm/open-match/values.yaml#L16
[om-example-chart-yaml-version]: https://github.com/googleforgames/open-match/blob/master/install/helm/open-match-example/Chart.yaml#L16
[om-example-values-yaml-version]: https://github.com/googleforgames/open-match/blob/master/install/helm/open-match-example/values.yaml#L16
[om-release]: https://github.com/googleforgames/open-match/releases/new
[readme-deploy]: https://github.com/googleforgames/open-match/blob/master/README.md#deploy-to-kubernetes
