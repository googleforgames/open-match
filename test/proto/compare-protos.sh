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

docker build -t gen-proto -f ./test/proto/Dockerfile --target compare .
if [ $? -ne 0 ]; then
    docker run --rm gen-proto
    printf "\nGenerated proto files are not updated. Please run 'make gen-proto'\n"
    exit 1
fi

docker run --rm gen-proto
if [ $? -ne 0 ]; then
    printf "\nGenerated proto files are not updated. Please run 'make gen-proto'\n"
    exit 1
fi

printf "\nGenerated proto files are up-to-date ╭(◔ ◡ ◔)/!\n"
