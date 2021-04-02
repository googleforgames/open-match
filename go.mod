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
go 1.14

require (
	contrib.go.opencensus.io/exporter/jaeger v0.2.1
	contrib.go.opencensus.io/exporter/ocagent v0.7.0
	contrib.go.opencensus.io/exporter/prometheus v0.2.0
	contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/Bose/minisentinel v0.0.0-20200130220412-917c5a9223bb
	github.com/TV4/logrus-stackdriver-formatter v0.1.0
	github.com/alicebob/miniredis/v2 v2.14.1
	github.com/aws/aws-sdk-go v1.35.26 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-redsync/redsync/v4 v4.0.3
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/protobuf v1.4.3
	github.com/gomodule/redigo v2.0.1-0.20191111085604-09d84710e01a+incompatible
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.8.0
	github.com/rs/xid v1.2.1
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/afero v1.4.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	go.opencensus.io v0.22.5
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	golang.org/x/sys v0.0.0-20201112073958-5cba982894dd // indirect
	golang.org/x/text v0.3.4 // indirect
	google.golang.org/api v0.35.0 // indirect
	google.golang.org/genproto v0.0.0-20201112120144-2985b7af83de
	google.golang.org/grpc v1.33.2
	google.golang.org/protobuf v1.25.0
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/api v0.0.0-20191004102349-159aefb8556b // kubernetes-1.14.10
	k8s.io/apimachinery v0.0.0-20191004074956-c5d2f014d689 // kubernetes-1.14.10
	k8s.io/client-go v11.0.1-0.20191029005444-8e4128053008+incompatible // kubernetes-1.14.10
	k8s.io/klog v1.0.0 // indirect
	k8s.io/utils v0.0.0-20200729134348-d5654de09c73 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)
