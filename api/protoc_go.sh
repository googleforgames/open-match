#!/bin/bash
# Script to compile golang versions of the OM proto files
#
# Copyright 2018 Google LLC
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
 
cd $GOPATH/src
protoc \
${GOPATH}/src/github.com/GoogleCloudPlatform/open-match/api/protobuf-spec/backend.proto \
${GOPATH}/src/github.com/GoogleCloudPlatform/open-match/api/protobuf-spec/frontend.proto \
${GOPATH}/src/github.com/GoogleCloudPlatform/open-match/api/protobuf-spec/mmlogic.proto \
${GOPATH}/src/github.com/GoogleCloudPlatform/open-match/api/protobuf-spec/messages.proto \
-I ${GOPATH}/src/github.com/GoogleCloudPlatform/open-match/ \
--go_out=plugins=grpc:$GOPATH/src
cd -
