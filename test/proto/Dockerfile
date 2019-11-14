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

######################################################################
# This target installs required protoc binaries; then generates files.
FROM golang:1.13.4-alpine3.10 as generated
ENV GO111MODULE=on
ENV CGO_ENABLED 0
ENV PROTOC_VERSION=3.8.0

# https://pkgs.alpinelinux.org/packages?name=protobuf&branch=v3.10
RUN apk add --no-cache protobuf=3.6.1-r1 git=2.22.0-r0 unzip=6.0-r4

# Install protoc
WORKDIR /
RUN wget -O protoc.zip https://github.com/google/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip
RUN unzip protoc.zip -d protoc
RUN cp -R /protoc/include /usr/local/include

# Install binaries
WORKDIR /usr/local/bin
RUN go build -i -pkgdir . github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
RUN go build -i -pkgdir . github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
RUN go build -i -pkgdir . github.com/golang/protobuf/protoc-gen-go
RUN go build -i -pkgdir . github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc

WORKDIR /proto
COPY api/ api/
COPY internal/api/*.proto internal/api/
COPY third_party/ third_party/
RUN mkdir -p open-match.dev/open-match/csharp/OpenMatch
# Generate files using the protos
# - swagger json files will sit within api/ directory
# - .pb. and .gw.pb files will sit within open-match.dev/open-match/pkg/pb and open-match.dev/open-match/internal/ipb directory
# - .cs files will sit within open-match.dev/open-match/csharp/OpenMatch directory
RUN protoc api/*.proto -I . -I /usr/local/include -I third_party/ \
  --plugin=protoc-gen-grpc=grpc_csharp_plugin \
  --grpc-gateway_out=logtostderr=true,allow_delete_body=true:. \
  --swagger_out=logtostderr=true,allow_delete_body=true:. \
  --go_out=plugins=grpc:. \
  --csharp_out=open-match.dev/open-match/csharp/OpenMatch
#  --doc_out=. \
#  --doc_opt=markdown,api.md
RUN protoc internal/api/*.proto -I . -I /usr/local/include -I third_party/ \
  --go_out=plugins=grpc:.

########################################################################
# This target compares the generated files with the ones in current HEAD
FROM generated as compare
WORKDIR /compare/head/
COPY api/ api/
COPY pkg/pb/ open-match.dev/open-match/pkg/pb
COPY internal/ipb open-match.dev/open-match/internal/ipb
COPY third_party/ third_party/
COPY --from=generated /proto /compare/tmp

WORKDIR /compare/tmp
CMD diff -q -r /compare/head/open-match.dev/open-match/internal/ /compare/tmp/open-match.dev/open-match/internal/
