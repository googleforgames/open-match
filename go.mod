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

// When updating Go version, update Dockerfile.ci, Dockerfile.base-build, and go.mod
go 1.16

require (
	contrib.go.opencensus.io/exporter/jaeger v0.2.1
	contrib.go.opencensus.io/exporter/ocagent v0.7.0
	contrib.go.opencensus.io/exporter/prometheus v0.2.0
	contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/Bose/minisentinel v0.0.0-20200130220412-917c5a9223bb
	github.com/TV4/logrus-stackdriver-formatter v0.1.0
	github.com/alicebob/miniredis/v2 v2.14.1
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-redsync/redsync/v4 v4.3.0
	github.com/golang/protobuf v1.5.2
	github.com/gomodule/redigo v2.0.1-0.20191111085604-09d84710e01a+incompatible
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.15.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.8.0
	github.com/rs/xid v1.2.1
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.8.1
	go.opencensus.io v0.24.0
	golang.org/x/net v0.3.0
	golang.org/x/sync v0.1.0
	google.golang.org/genproto v0.0.0-20221207170731-23e4bf6bdc37
	google.golang.org/grpc v1.51.0
	google.golang.org/protobuf v1.28.1
	k8s.io/api v0.23.4 // kubernetes-1.14.10
	k8s.io/apimachinery v0.23.4 // kubernetes-1.14.10
	k8s.io/client-go v0.23.4 // kubernetes-1.14.10
)

require (
	github.com/aws/aws-sdk-go v1.35.26 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/spf13/afero v1.4.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)
