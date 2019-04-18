# Release {version}

<!--
This is the release issue template. Make a copy of the markdown in this page
and copy it into a release issue. Fill in relevent values, found inside {}
{version} should be replaced with the version ie: 0.5.0.

There are 3 types of releases:
 * Release Candidates - 1.0.0-rc1
 * Full Releases - 1.2.0
 * Hot Fixes - 1.0.1

# Release Candidate and Full Release Process

1. Create a Release Issue from the [release issue template](./templates/release_issue.md).
1. Label the issue `kind/release`, and attach it to the milestone that it matches.
1. Complete all items in the release issue checklist.
1. Close the release issue.

# Hot Fix Process
 
1. Hotfixes will occur as needed, to be determined by those will commit access on the repository.
1. Create a Release Issue from the [release issue template](./templates/release_issue.md).
1. Label the issue `kind/release`, and attach it to the next upcoming milestone.
1. Complete all items in the release issue checklist.
1. Close the release issue.

!-->
Complete Milestone
------------------
- [ ] Create the next version milestone, use [semantic versioning](https://semver.org/) when naming it to be consistent with the [Go community](https://blog.golang.org/versioning-proposal).
- [ ] Visit the [milestone](https://github.com/GoogleCloudPlatform/open-match/milestone).
  - [ ] Open a document for a draft [release notes](release.md).
  - [ ] Add the milestone tag to all PRs and issues that were merged since the last milestone. Look at the [releases page](https://github.com/GoogleCloudPlatform/open-match/releases) and look for the "X commits to master since this release" for the diff. The link resolves to, https://github.com/GoogleCloudPlatform/open-match/compare/v{version}...master.
  - [ ] Review all [milestone-less closed issues](https://github.com/GoogleCloudPlatform/open-match/issues?q=is%3Aissue+is%3Aclosed+no%3Amilestone) and assign the appropriate milestone.
  - [ ] Review all [issues in milestone](https://github.com/GoogleCloudPlatform/open-match/milestones) for proper [labels](https://github.com/GoogleCloudPlatform/open-match/labels) (ex: area/build).
  - [ ] Review all [milestone-less closed PRs](https://github.com/GoogleCloudPlatform/open-match/pulls?q=is%3Apr+is%3Aclosed+no%3Amilestone) and assign the appropriate milestone.
  - [ ] Review all [PRs in milestone](https://github.com/GoogleCloudPlatform/open-match/milestones) for proper [labels](https://github.com/GoogleCloudPlatform/open-match/labels) (ex: area/build).
  - [ ] View all open entries in milestone and move them to a future milestone if they aren't getting closed in time. https://github.com/GoogleCloudPlatform/open-match/milestones/v{version}
  - [ ] Review all closed PRs against the milestone. Put the user visible changes into the release notes using the suggested format. https://github.com/GoogleCloudPlatform/open-match/pulls?q=is%3Apr+is%3Aclosed+milestone%3Av{version}
  - [ ] Review all closed issues against the milestone. Put the user visible changes into the release notes using the suggested format. https://github.com/GoogleCloudPlatform/open-match/issues?utf8=%E2%9C%93&q=is%3Aissue+is%3Aclosed+milestone%3Av{version}
  - [ ] Verify the [milestone](https://github.com/GoogleCloudPlatform/open-match/milestones) is effectively 100% at this point with the exception of the release issue itself.

TODO: Add details for appropriate tagging for issues.

Build Artifacts
---------------
- [ ] Create a PR to bump the version.
  - [ ] Open the [`Makefile`](makefile-version) and change BASE_VERSION value. Release candidates use the -rc# suffix.
  - [ ] Open the [`install/helm/open-match/Chart.yaml`](om-chart-yaml-version) and [`install/helm/open-match-example/Chart.yaml`](om-example-chart-yaml-version) and change the `appVersion` and `version` entries.
  - [ ] Open the [`install/helm/open-match/values.yaml`](om-values-yaml-version) and [`install/helm/open-match-example/values.yaml`](om-example-values-yaml-version) and change the `tag` entries.
  - [ ] Open the [`site/config.toml`]  and change the `release_branch` and `release_version` entries.
  - [ ] Open the [`README.md`](readme-deploy) update the version references.
  - [ ] Run `make clean release`
  - [ ] There might be additional references to the old version but be careful not to change it for places that have it for historical purposes.
  - [ ] Submit the pull request.
- [ ] Take note of the git hash in master, `git checkout master && git pull master && git rev-parse HEAD`
- [ ] Go to [Cloud Build](https://pantheon.corp.google.com/cloud-build/triggers?project=open-match-build), under Post Submit click "Run Trigger".
- [ ] Go to the History section and find the "Post Submit" build that's running. Wait for it to go Green. If it's red fix error repeat this section. Take note of version tag for next step.
- [ ] Run `./docs/governance/templates/release.sh {source version tag} {version}` to copy the images to open-match-public-images.
- [ ] Create a *draft* release with the [release template][release-template]
  - [ ] Make a `tag` with the release version. The tag must be v{version}. Example: v0.5.0. Append -rc# for release candidates. Example: v0.5.0-rc1.
  - [ ] Copy the files from `build/release/` generated from `make release` from earlier as release artifacts.
- [ ] Run `make delete-gke-cluster create-gke-cluster push-helm sleep-10 install-chart install-example-chart` and verify that the pods are all healthy.
- [ ] Run `make delete-gke-cluster create-gke-cluster` and run through the instructions under the [README](readme-deploy), verify the pods are healthy. You'll need to adjust the path to the `install/yaml/install.yaml` and `install/yaml/install-example.yaml` in your local clone since you haven't published them yet.
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
