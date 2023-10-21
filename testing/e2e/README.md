# Open Match E2E Tests

Open Match can run locally (aka Minimatch) or on a Kubernetes cluster. This package provides a framework to
create tests that can run in both environments.

# Minimatch
Minimatch is a single process instance of Open Match that runs with a test in-memory Redis instance.
This allows you to run Open Match tests without a Kubernetes cluster quickly. There's only 1 instance of
each service and they are served from the same port on localhost.

# Kubernetes
Running Open Match on Kubernetes is how Open Match is meant to be run in production. By default, 3
replicas of each server is running behind a load balancer. Running tests against a cluster has the benefits
of running the tests in a realistic setting where requests may not be routed to the same server.

# How to Use the E2E Test Framework?

For a new test package under test/e2e/ do the following:

Copy the contents of testing/e2e/main_test.go and change the package name and add
the `open-match.dev/open-match/testing/e2e` import.

Example (may be out of date):
```golang
// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tickets

import (
	"open-match.dev/open-match/testing/e2e"
	"testing"
)

func TestMain(m *testing.M) {
	e2e.RunMain(m)
}
```

Then create your tests like other tests in the `test/e2e/` directory.

If your test is not compatible with E2E cluster tests then add
`// +build !e2ecluster` at the top of the file. It must be the first line!

# How Does it Work?

## Minimatch

Minimatch mode essentially binds all the services that Open Match provides
into a single gRPC/HTTP server. The magic happens in the `internal/app/minimatch/minimatch.go`'s
`BindService()` method.

In addition, there's an in-memory Redis instance that's initialized and bound in `testing/e2e/in_memory.go`.

From here the `OM` instance encapsulates the details of communicating with these components.

## Kubernetes Cluster

Kubernetes cluster mode is managed via `testing/e2e/cluster.go`
