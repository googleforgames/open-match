#!/bin/bash

# Copyright 2019 Google LLC
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#     http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

docker build -t gen-proto -f ./test/proto/Dockerfile --target generated .
docker create -ti --name dummy-gen-proto gen-proto bash
docker cp dummy-gen-proto:/proto/api .
docker cp dummy-gen-proto:/proto/open-match.dev/open-match/internal/ipb ./internal/
docker cp dummy-gen-proto:/proto/open-match.dev/open-match/pkg/pb ./pkg/
docker rm -f dummy-gen-proto