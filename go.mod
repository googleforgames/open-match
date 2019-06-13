module open-match.dev/open-match

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

go 1.12

require (
	contrib.go.opencensus.io/exporter/jaeger v0.1.0
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	contrib.go.opencensus.io/exporter/stackdriver v0.11.0
	contrib.go.opencensus.io/exporter/zipkin v0.1.1
	github.com/TV4/logrus-stackdriver-formatter v0.1.0
	github.com/alicebob/gopher-json v0.0.0-20180125190556-5a6b3ba71ee6 // indirect
	github.com/alicebob/miniredis v2.5.0+incompatible
	github.com/aws/aws-sdk-go v1.19.25 // indirect
	github.com/cenkalti/backoff v2.1.1+incompatible
	github.com/fsnotify/fsnotify v1.4.7
	github.com/gogo/protobuf v1.2.1
	github.com/golang/protobuf v1.3.1
	github.com/gomodule/redigo v1.7.1-0.20190322064113-39e2c31b7ca3
	github.com/grpc-ecosystem/grpc-gateway v1.8.5
	github.com/openzipkin/zipkin-go v0.1.6
	github.com/pkg/errors v0.8.0
	github.com/prometheus/client_golang v0.9.3-0.20190127221311-3c4408c8b829
	github.com/rs/xid v1.2.1
	github.com/sirupsen/logrus v1.4.1
	github.com/spf13/viper v1.3.2
	github.com/stretchr/testify v1.3.0
	github.com/yuin/gopher-lua v0.0.0-20190514113301-1cd887cd7036 // indirect
	go.opencensus.io v0.21.0
	google.golang.org/genproto v0.0.0-20190404172233-64821d5d2107
	google.golang.org/grpc v1.20.0
)
