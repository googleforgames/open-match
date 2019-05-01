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

// Package harness is the evaluator harness contains common scaffolding needed by the Evaluator.
// The harness currently runs the evaluator loop that periodically scans the database
// for new proposals and calls the user defined evaluation function with the
// collection of proposals. The harness accepts the approved matches from the user
// and modifies the database to indicate these results.
// Currently, the harness logic simply abstracts the user from the database
// concepts. In future, as the proposals move out of the database, this harness
// will be a GRPC service that accepts proposal stream from Open Match and
// calls user's evaluation logic with this collection of proposals. Thus the
// harness will minimize impact of these future changes on user's evaluation function.
package harness
