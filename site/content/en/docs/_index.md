
---
title: "Open Match"
linkTitle: "Documentation"
weight: 20
menu:
  main:
    weight: 20
---

Open Match is an open source game matchmaking framework designed to allow game creators to build matchmakers of any size easily and with as much possibility for sharing and code re-use as possible. It’s designed to be flexible (run it anywhere Kubernetes runs), extensible (match logic can be customized to work for any game), and scalable.

Matchmaking is a complicated process, and when large player populations are involved, many popular matchmaking approaches touch on significant areas of computer science including graph theory and massively concurrent processing. Open Match is an effort to provide a foundation upon which these difficult problems can be addressed by the wider game development community. As Josh Menke &mdash; famous for working on matchmaking for many popular triple-A franchises &mdash; put it:

["Matchmaking, a lot of it actually really is just really good engineering. There's a lot of really hard networking and plumbing problems that need to be solved, depending on the size of your audience."](https://youtu.be/-pglxege-gU?t=830)

This project attempts to solve the networking and plumbing problems, so game developers can focus on the logic to match players into great games.

## Disclaimer
This software is currently alpha, and subject to change.  Although Open Match has already been used to run [production workloads within Google](https://cloud.google.com/blog/topics/inside-google-cloud/no-tricks-just-treats-globally-scaling-the-halloween-multiplayer-doodle-with-open-match-on-google-cloud), but it's still early days on the way to our final goal. There's plenty left to write and we welcome contributions. **We strongly encourage you to engage with the community through the [Slack or Mailing lists](#get-involved) if you're considering using Open Match in production before the 1.0 release, as the documentation is likely to lag behind the latest version a bit while we focus on getting out of alpha/beta as soon as possible.**

## Version
[The current stable version in master is 0.3.1 (alpha)](https://github.com/GoogleCloudPlatform/open-match/releases/tag/v0.3.1-alpha).  At this time only bugfixes and doc update pull requests will be considered.
Version 0.4.0 is in active development; please target code changes to the 040wip branch.
