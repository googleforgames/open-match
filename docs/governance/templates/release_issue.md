# Release {version}

<!--
This is the release issue template. Make a copy of the markdown in this page
and copy it into a release issue. Fill in relevent values, found inside {}
{version} should be replaced with the version ie: 0.5.0.

There are 3 types of releases:
 * Release Candidates - 1.0.0-rc.1
 * Full Releases - 1.2.0
 * Hot Fixes - 1.0.1

# Release Candidate and Full Release Process

1. Create a Release Issue from the [release issue template](./release_issue.md).
1. Label the issue `kind/release`, and attach it to the milestone that it matches.
1. Complete all items in the release issue checklist.
1. Close the release issue.

# Hot Fix Process

1. Hotfixes will occur as needed, to be determined by those will commit access on the repository.
1. Create a Release Issue from the [release issue template](./release_issue.md).
1. Label the issue `kind/release`, and attach it to the next upcoming milestone.
1. Complete all items in the release issue checklist.
1. Close the release issue.

!-->

Open Match Release Process
==========================

These instructions are the steps needed to create a release of Open Match. All of these steps will be done in a `release-X.Y` branch. **No changes will be submitted to the master branch unless explicitly stated.**

Create a Branch
---------------
**These instructions are for the first release candidate of a minor (X.Y) release.**
- [ ] Create a branch in the upstream repository. It should be named release-X.Y. Example: release-0.5. At this point there's effectively a code freeze for this version and all work on master will be included in a future version.

Create New Version
------------------
_Find the release branch relevant for your release. It will have the name release-X.Y. While doing a release in the branch no changes should be submitted outside of the process. Also all changes done _

- [ ] Announce a PR freeze on release-X.Y branch on [open-match-discuss@](mailing-list-post).
- [ ] Create a PR to bump the version. Release candidates use the -rc.# suffix.
  - [ ] Open the [`Makefile`](makefile-version) and change BASE_VERSION entry.
  - [ ] Open the [`install/helm/open-match/Chart.yaml`](om-chart-yaml-version) and [`install/helm/open-match-example/Chart.yaml`](om-example-chart-yaml-version) and change the `appVersion` and `version` entries.
  - [ ] Open the [`install/helm/open-match/values.yaml`](om-values-yaml-version) and [`install/helm/open-match-example/values.yaml`](om-example-values-yaml-version) and change the `tag` entries.
  - [ ] Open the [`site/config.toml`] and change the `release_branch` and `release_version` entries.
  - [ ] Open the [`cloudbuild.yaml`] and change the `_OM_VERSION` entry.
  - [ ] Run `make clean release`
  - [ ] There might be additional references to the old version but be careful not to change it for places that have it for historical purposes.
  - [ ] Submit the pull request.

Complete Milestone
------------------
- [ ] Create the next version milestone, use [semantic versioning](https://semver.org/) when naming it to be consistent with the [Go community](https://blog.golang.org/versioning-proposal).
- [ ] Visit the [milestone](https://github.com/GoogleCloudPlatform/open-match/milestone).
  - [ ] Create a *draft* release with the [release template](release-template).
    - [ ] `Tag` = v{version}. Example: v0.5.0. Append -rc.# for release candidates. Example: v0.5.0-rc.1.
    - [ ] `Target` = release-X.Y. Example: release-0.5.
    - [ ] `Release Title` = `Tag`
    - [ ] `Write` section will contain the contents from the [release template](release-template).
  - [ ] Add the milestone to all PRs and issues that were merged since the last milestone. Look at the [releases page](https://github.com/GoogleCloudPlatform/open-match/releases) and look for the "X commits to master since this release" for the diff.
  - [ ] Review all [milestone-less closed issues](https://github.com/GoogleCloudPlatform/open-match/issues?q=is%3Aissue+is%3Aclosed+no%3Amilestone) and assign the appropriate milestone.
  - [ ] Review all [issues in milestone](https://github.com/GoogleCloudPlatform/open-match/milestones) for proper [labels](https://github.com/GoogleCloudPlatform/open-match/labels) (ex: area/build).
  - [ ] Review all [milestone-less closed PRs](https://github.com/GoogleCloudPlatform/open-match/pulls?q=is%3Apr+is%3Aclosed+no%3Amilestone) and assign the appropriate milestone.
  - [ ] Review all [PRs in milestone](https://github.com/GoogleCloudPlatform/open-match/milestones) for proper [labels](https://github.com/GoogleCloudPlatform/open-match/labels) (ex: area/build).
  - [ ] View all open entries in milestone and move them to a future milestone if they aren't getting closed in time. https://github.com/GoogleCloudPlatform/open-match/milestones/v{version}
  - [ ] Review all closed PRs against the milestone. Put the user visible changes into the release notes using the suggested format. https://github.com/GoogleCloudPlatform/open-match/pulls?utf8=%E2%9C%93&q=is%3Apr+is%3Aclosed+is%3Amerged+milestone%3Av{version}
  - [ ] Review all closed issues against the milestone. Put the user visible changes into the release notes using the suggested format. https://github.com/GoogleCloudPlatform/open-match/issues?utf8=%E2%9C%93&q=is%3Aissue+is%3Aclosed+milestone%3Av{version}
  - [ ] Verify the [milestone](https://github.com/GoogleCloudPlatform/open-match/milestones) is effectively 100% at this point with the exception of the release issue itself.

TODO: Add guidelines for labeling issues.

Build Artifacts
---------------

- [ ] Go to [Cloud Build](https://pantheon.corp.google.com/cloud-build/triggers?project=open-match-build), under Post Submit click "Run Trigger".
- [ ] Go to the History section and find the "Post Submit" build that's running. Wait for it to go Green. If it's red, fix error repeat this section. Take note of the docker image version tag for next step. Example: 0.5.0-a4706cb.
- [ ] Run `./docs/governance/templates/release.sh {source version tag} {version}` to copy the images to open-match-public-images.
- [ ] If this is a new minor version in the newest major version then run `./docs/governance/templates/release.sh {source version tag} latest`.
- [ ] Copy the files from `build/release/` generated from `make release` to the release draft you created.
- [ ] Run `make REGISTRY=gcr.io/open-match-public-images TAG={version} delete-gke-cluster create-gke-cluster push-helm sleep-10 install-chart install-example-chart` and verify that the pods are all healthy.
- [ ] Run `make delete-gke-cluster create-gke-cluster` and run through the instructions under the [README](readme-deploy), verify the pods are healthy. You'll need to adjust the path to the `build/release/install.yaml` and `build/release/install-example.yaml` in your local clone since you haven't published them yet.
- [ ] Open the [`README.md`](readme-deploy) update the version references and submit. (Release candidates can ignore this step.)
- [ ] Publish the [Release](om-release) in Github.

Announce
--------
- [ ] Send an email to the [mailing list](mailing-list-post) with the release details (copy-paste the release blog post)
- [ ] Send a chat on the [Slack channel](om-slack). "Open Match {version} has been released! Check it out at {release url}."

[om-slack]: https://open-match.slack.com/
[mailing-list-post]: https://groups.google.com/forum/#!newtopic/open-match-discuss
[release-template]: https://github.com/GoogleCloudPlatform/open-match/blob/master/docs/governance/templates/release.md
[makefile-version]: https://github.com/GoogleCloudPlatform/open-match/blob/master/Makefile#L53
[om-example-chart-yaml-version]: https://github.com/GoogleCloudPlatform/open-match/blob/master/install/helm/open-match/Chart.yaml#L16
[om-example-values-yaml-version]: https://github.com/GoogleCloudPlatform/open-match/blob/master/install/helm/open-match/values.yaml#L16
[om-example-chart-yaml-version]: https://github.com/GoogleCloudPlatform/open-match/blob/master/install/helm/open-match-example/Chart.yaml#L16
[om-example-values-yaml-version]: https://github.com/GoogleCloudPlatform/open-match/blob/master/install/helm/open-match-example/values.yaml#L16
[om-release]: https://github.com/GoogleCloudPlatform/open-match/releases/new
[readme-deploy]: https://github.com/GoogleCloudPlatform/open-match/blob/master/README.md#deploy-to-kubernetes
