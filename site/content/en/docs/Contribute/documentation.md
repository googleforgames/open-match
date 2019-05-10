---
title: "Documentation Editing and Contribution"
linkTitle: "Edit the Docs"
weight: 1000
description: >
  How to contribute to the documentation.
---

Editing the documentation is pretty simple if your familiar with [Markdown](https://github.com/adam-p/markdown-here/wiki/Markdown-Cheatsheet).

## Simple Edits

If you want to make simple edits to a page (changes that do not require formatting or images) and you're not an approver.

 1. Click the `Edit this page` link at the top right of the documentation page you want to edit.
 1. You'll be redirected to an editor, make your changes there and click `Propose file change`.
 1. Your changes will be reviewed, make any adjustments and submit.

## Advanced Edits

If you're adding a new page, changing the landing page, or making substantial changes.

 1. Checkout the repository.
 1. Create a branch via `git checkout -b mychange upstream/master`.
 1. Run `make run-site` to start the webserver. It'll be hosted on http://localhost:8080/.
 1. Open your favorite editor and navigate to the documentation which is stored at `$(REPOSITORY_ROOT)/site/content/en/`.
 1. Make your changes and run `git commit -m` and `git push`
 1. Follow the Github Pull Request workflow to submit your changes.

## Platform

This site and documentation is built with a combination of Hugo, static site generator,
with the Docsy theme for open source documentation.

- [Hugo Documentation](https://gohugo.io/documentation/)
- [Docsy Guide](https://github.com/google/docsy)
- [Link Checker](https://github.com/wjdp/htmltest)
