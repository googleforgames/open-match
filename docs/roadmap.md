# Open Match Roadmap

Open Match is currently at release 0.4.0. Open Match 0.5.0 currently has a Release Candidate and we are targeting to cut the release on 04/25/2019.

Releases can be found on the [releases page](https://github.com/googleforgames/open-match/releases).

Below sections detail the themes and the roadmap for the future releases. The tasks listed for the 0.6.0 release have been finalized and are well understood. As for the 0.7.0 and beyond, the tasks currently identified are listed. These are subject to change as we make our way through 0.6.0 release and get more feedback from the community.

## 0.5.0 - Usability

The primary focus of the 0.5 release is usability. The goal for this release is to make Open Match easy to build and deploy and have solid supporting documentation. Users should be able to try Open Match 0.5.0 functionality and experiment with its features, MMFs etc. Here are some planned features for this release:

- [X] Add support to invoke MMFs as a gRPC function call.
- [X] Provide a gRPC serving harness and an example MMF built using this harness. (golang based).
- [X] Provide a evaluation harness and a sample evaluator using this harness (golang based)
- [X] Deprecate the k8s based job scheduling mechanism for MMFs, Evaluator in favor of hosted MMFs, Evaluator.
- [X] Switch all core Open Match services to use gRPC style request / response protos.
- [X] Documentation: Add basic user, developer documentation and set up the Open Match website.
- [X] Create and document a formal release process.
- [X] Improve developer experience (simplify compiling, deploying and validating)

## 0.6.0 - API changes, Maturity

In 0.6.0 release, we are revisiting the Data Model and the API surface exposed by Open Match. The goal of this release is to front-load a major API refactoring that will facilitate achieving scale and other productionizing goals in forthcoming releases. Although breaking chagnes can happen any time till 1.0, the goal is to implement any major breaking changes in 0.6.0 so that future chagnes if any are relatively minor. Customer should be able to start building their Match Makers using the 0.6.0 API surface.

Here are the tasks planned for 0.6.0 release:

- [ ] Implement the new Data model and the API changes for the Frontend, Backend and MMLogic API [Change Proposal](https://github.com/googleforgames/open-match/issues/279)
- [ ] Accept multiple proposals per MMF execution.
- [ ] Remove persistance of matches and proposals from Open Match state storage.
- [ ] Implement synchronized evaluation to eliminate use of state storage during evaluation.
- [ ] Introduce test framework for unit testing, Component testing and E2E testing.
- [ ] Add unit tests, component tests and integration tests for Open Match core components and examples.
- [ ] Update harness, evaluator, mmf samples etc., to reflect the API changes.
- [ ] Update documentation, website to reflect 0.6.0 API changes.

## 0.7.0 - Scale, Operationalizing

Features for 0.7.0 are targeted to enable Open Match to be productionizable. Note that as we identify more feature work past 0.6.0, these tasks may get pushed to future releases. However, these are core tasks that need to be addressed before Open Match reaches 1.0.

- [ ] Introduce Test framework for load, performance testing
- [ ] Automated Load / performance / scale tests
- [ ] Test results Dashboard
- [ ] Add support for Instrumentation, Monitoring, Dashboards
- [ ] Add support or Metrics collection, Analytics, Dashboards
- [ ] Identify Autoscaling patterns for each component and configure them.

## Other Features

Below are additional features that are not tied to a specific release but will be added incrementally across releases:

- [ ] Harness support for Python, PHP, C#, C++
- [ ] User Guide for Open Match, Tutorials
- [ ] Developer Guide for Open Match
- [ ] APIs & Reference
- [ ] Concept Documentation
- [ ] Website Improvements

## 1.1.0

Below are the features that have been identified but are not considered critical for Open Match (as a match making framework) to itself reach 1.0. Any other features that are related to Open Match ecosystem but not a part of the framework itself can be listed here. These features may not necessarily wait for Open Match 1.0 and can be implemented before that - but any of the currently identified 1.0 tasks are higher in priority than these to make Open Match production ready.

- [ ] Canonical usable examples out of box.
- [ ] KNative support to run MMFs
- [ ] OSS Director to integrate with Agones, other DGS backends

### Special Thanks
- Thanks to https://jbt.github.io/markdown-editor/ for help in marking this document down.
