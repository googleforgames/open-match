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

// Package golang contains the go files required to run the harness as a GRPC
// service. To use this harness, you should author the match making function and
// pass that in as the callback when setting up the function harness service.
// Note that the main package for the harness does very little except read the
// config and set up logging and metrics, then start the server. The harness
// functionality is implemented in harness.go, which implements the gRPC server
// defined in the pkg/pb/matchfunction.pb.go file.
package golang
