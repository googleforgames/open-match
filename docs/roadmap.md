# Roadmap.  [Subject to change]
Releases are scheduled for every 6 weeks.  **Every release is a stable, long-term-support version**.  Even for alpha releases, best-effort support is available. With a little work and input from an experienced live services developer, you can go to production with any version on the [releases page](https://github.com/GoogleCloudPlatform/open-match/releases). 

Our current thinking is to wait to take Open Match out of alpha/beta (and label it 1.0) until it can be used out-of-the-box, standalone, for developers that don’t have any existing platform services.  Which is to say, the majority of **established game developers likely won't have any reason to wait for the 1.0 release if Open Match already handles your needs**. If you already have live platform services that you plan to integrate Open Match with (player authentication, a group invite system, dedicated game servers, metrics collection, logging aggregation, etc), then a lot of the features planned between 0.4.0 and 1.0 likely aren't of much interest to you anyway.

## Upcoming releases
* 0.4.0 - Agones Integration & MMF on [Knative](https://cloud.google.com/Knative/)
    MMF instrumentation
    Match object expiration / lazy deletion
    API autoscaling by default
    API changes after this will likely be additions or very minor
* 0.5.0 - Tracing, Metrics, and KPI Dashboard
* 0.6.0 - Load testing suite
* 1.0.0 - API Formally Stable.  Breaking API changes will require a new major version number.
* 1.1.0 - Canonical MMFs

## Philosophy
* The next version (0.4.0) will focus on making MMFs run on serverless platforms - specifically Knative. This will just be first steps, as Knative is still pretty early.  We want to get a proof of concept working so we can roadmap out the future "MMF on Knative" experience.  Our intention is to keep MMFs as compatible as possible with the current Kubernetes job-based way of doing them.  Our hope is that by the time Knative is mature, we’ll be able to provide a [Knative build](https://github.com/Knative/build) pipeline that will take existing MMFs and build them as Knative functions.  In the meantime, we’ll map out a relatively painless (but not yet fully automated) way to make an existing MMF into a Kubernetes Deployment that looks as similar to what [Knative serving](https://github.com/knative/serving) is shaping up to be, in an effort to make the eventual switchover painless. Basically all of this is just _optimizing MMFs to make them spin up faster and take less resources_, **we're not planning to change what MMFs do or the interfaces they need to fulfill**.  Existing MMFs will continue to run as-is, and in the future moving them to Knative should be both **optional** and **largely automated**.
* 0.4.0 represents the natural stopping point for adding new functionality until we have more community uptake and direction. We don't anticipate many API changes in 0.4.0 and beyond.  Maybe new API calls for new functionality, but we're unlikely to see big shifts in existing calls through 1.0 and its point releases.  We'll issue a new major release version if we decide we need those changes.
* The 0.5.0 version and beyond will be focused on operationalizing the out-of-the-box experience. Metrics and analytics and a default dashboard, additional tooling, and a load testing suite are all planned.  We want it to be easy for operators to see KPI and know what's going on with Open Match. 
