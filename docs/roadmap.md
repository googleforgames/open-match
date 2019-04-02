# Roadmap.  [Subject to change]
Releases are scheduled for every 6 weeks.  **Every release is a stable, long-term-support version**.  Even for alpha releases, best-effort support is available. With a little work and input from an experienced live services developer, you can go to production with any version on the [releases page](https://github.com/GoogleCloudPlatform/open-match/releases). 

Our current thinking is to wait to take Open Match out of alpha/beta (and label it 1.0) until it can be used out-of-the-box, standalone, for developers that don’t have any existing platform services.  Which is to say, the majority of **established game developers likely won't have any reason to wait for the 1.0 release if Open Match already handles your needs**. If you already have live platform services that you plan to integrate Open Match with (player authentication, a group invite system, dedicated game servers, metrics collection, logging aggregation, etc), then a lot of the features planned between 0.4.0 and 1.0 likely aren't of much interest to you anyway.

## Upcoming releases
* **0.4.0** &mdash; Agones Integration & MMF on [Knative](https://cloud.google.com/knative/)
    MMF instrumentation
    Match object expiration / lazy deletion
    API autoscaling by default
    API changes after this will likely be additions or very minor
* **0.5.0** &mdash; Tracing, Metrics, and KPI Dashboard
* **0.6.0** &mdash; Load testing suite
* **1.0.0** &mdash; API Formally Stable.  Breaking API changes will require a new major version number.
* **1.1.0** &mdash; Canonical MMFs

## Philosophy
* The next version (0.4.0) will focus on making MMFs run on serverless platforms - specifically Knative. This will just be first steps, as Knative is still pretty early.  We want to get a proof of concept working so we can roadmap out the future "MMF on Knative" experience.  Our intention is to keep MMFs as compatible as possible with the current Kubernetes job-based way of doing them.  Our hope is that by the time Knative is mature, we’ll be able to provide a [Knative build](https://github.com/Knative/build) pipeline that will take existing MMFs and build them as Knative functions.  In the meantime, we’ll map out a relatively painless (but not yet fully automated) way to make an existing MMF into a Kubernetes Deployment that looks as similar to what [Knative serving](https://github.com/knative/serving) is shaping up to be, in an effort to make the eventual switchover painless. Basically all of this is just _optimizing MMFs to make them spin up faster and take less resources_, **we're not planning to change what MMFs do or the interfaces they need to fulfill**.  Existing MMFs will continue to run as-is, and in the future moving them to Knative should be both **optional** and **largely automated**.
* 0.4.0 represents the natural stopping point for adding new functionality until we have more community uptake and direction. We don't anticipate many API changes in 0.4.0 and beyond.  Maybe new API calls for new functionality, but we're unlikely to see big shifts in existing calls through 1.0 and its point releases.  We'll issue a new major release version if we decide we need those changes.
* The 0.5.0 version and beyond will be focused on operationalizing the out-of-the-box experience. Metrics and analytics and a default dashboard, additional tooling, and a load testing suite are all planned.  We want it to be easy for operators to see KPI and know what's going on with Open Match. 

# Planned improvements
See the [provisional roadmap](docs/roadmap.md) for more information on upcoming releases.

## Documentation 
- [ ] “Writing your first matchmaker” getting started guide will be included in an upcoming version.
- [ ] Documentation for using the example customizable components and the `backendstub` and `frontendstub` applications to do an end-to-end (e2e) test will be written. This all works now, but needs to be written up.
- [ ] Documentation on release process and release calendar.

## State storage
- [X] All state storage operations should be isolated from core components into the `statestorage/` modules.  This is necessary precursor work to enabling Open Match state storage to use software other than Redis.
- [X] [The Redis deployment should have an example HA configuration](https://github.com/GoogleCloudPlatform/open-match/issues/41)
- [X] Redis watch should be unified to watch a hash and stream updates.  The code for this is written and validated but not committed yet. 
- [ ] We don't want to support two redis watcher code paths, but we will until golang protobuf reflection is a bit more usable. [Design doc](https://docs.google.com/document/d/19kfhro7-CnBdFqFk7l4_HmwaH2JT_Rhw5-2FLWLEGGk/edit#heading=h.q3iwtwhfujjx), [github issue](https://github.com/golang/protobuf/issues/364)
- [X] Player/Group records generated when a client enters the matchmaking pool need to be removed after a certain amount of time with no activity. When using Redis, this will be implemented as a expiration on the player record.

## Instrumentation / Metrics / Analytics
- [ ] Instrumentation of MMFs is in the planning stages.  Since MMFs are by design meant to be completely customizable (to the point of allowing any process that can be packaged in a Docker container), metrics/stats will need to have an expected format and formalized outgoing pathway.  Currently the thought is that it might be that the metrics should be written to a particular key in statestorage in a format compatible with opencensus, and will be collected, aggreggated, and exported to Prometheus using another process.
- [ ] [OpenCensus tracing](https://opencensus.io/core-concepts/tracing/) will be implemented in an upcoming version.  This is likely going to require knative.
- [X] Read logrus logging configuration from matchmaker_config.json.

## Security
- [ ] The Kubernetes service account used by the MMFOrc should be updated to have min required permissions. [Issue 52](issues/52)

## Kubernetes
- [ ] Autoscaling isn't turned on for the Frontend or Backend API Kubernetes deployments by default.
- [X] A [Helm](https://helm.sh/) chart to stand up Open Match may be provided in an upcoming version. For now just use the [installation YAMLs](./install/yaml).
- [ ] A knative-based implementation of MMFs is in the planning stages.

## CI / CD / Build
- [X] We plan to host 'official' docker images for all release versions of the core components in publicly available docker registries soon.  This is tracked in [Issue #45](issues/45) and is blocked by [Issue 42](issues/42).
- [X] CI/CD for this repo and the associated status tags are planned.
- [ ] Golang unit tests will be shipped in an upcoming version.
- [ ] A full load-testing and e2e testing suite will be included in an upcoming version.

## Will not Implement 
- [X] Defining multiple images inside a profile for the purposes of experimentation adds another layer of complexity into profiles that can instead be handled outside of open match with custom match functions in collaboration with a director (thing that calls backend to schedule matchmaking) 

### Special Thanks
- Thanks to https://jbt.github.io/markdown-editor/ for help in marking this document down.
