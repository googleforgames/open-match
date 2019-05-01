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

/*
FrontendAPI contains the unique files required to run the API endpoints for
Open Match's frontend. It is assumed you'll either integrate calls to these
endpoints directly into your game client (simple use case), or call these
endpoints from other, established platform services in your infrastructure
(more complicated use cases).

Note that the main package for frontendapi does very little except read the
config and set up logging and metrics, then start the server.  Almost all the
work is being done by frontendapi/apisrv, which implements the gRPC server
defined in the frontendapi/proto/frontend.pb.go file.
*/

// Package frontendapi provides the scaffolding for starting the service.
package frontendapi
