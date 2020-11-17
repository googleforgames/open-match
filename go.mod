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
	cloud.google.com/go v0.47.0 // indirect
	contrib.go.opencensus.io/exporter/jaeger v0.1.0
	contrib.go.opencensus.io/exporter/ocagent v0.6.0
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	contrib.go.opencensus.io/exporter/stackdriver v0.12.8
	github.com/Bose/minisentinel v0.0.0-20191213132324-b7726ed8ed71
	github.com/TV4/logrus-stackdriver-formatter v0.1.0
	github.com/alicebob/miniredis/v2 v2.11.0
	github.com/apache/thrift v0.13.0 // indirect
	github.com/aws/aws-sdk-go v1.25.27 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/fsnotify/fsnotify v1.4.7
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/groupcache v0.0.0-20191027212112-611e8accdfc9 // indirect
	github.com/golang/protobuf v1.3.2
	github.com/gomodule/redigo v2.0.1-0.20191111085604-09d84710e01a+incompatible
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0
	github.com/grpc-ecosystem/grpc-gateway v1.12.0
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/json-iterator/go v1.1.8 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/pelletier/go-toml v1.6.0 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.2.1
	github.com/pseudomuto/protoc-gen-doc v1.3.2 // indirect
	github.com/rs/xid v1.2.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/afero v1.2.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.5.0
	github.com/stretchr/testify v1.4.0
	go.opencensus.io v0.22.1
	golang.org/x/crypto v0.0.0-20191105034135-c7e5f84aec59 // indirect
	golang.org/x/net v0.0.0-20191105084925-a882066a44e0
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20191105231009-c1f44814a5cd // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	google.golang.org/api v0.13.0 // indirect
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/genproto v0.0.0-20191028173616-919d9bdd9fe6
	google.golang.org/grpc v1.25.0
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.2.5 // indirect
	k8s.io/api v0.0.0-20191004102255-dacd7df5a50b // kubernetes-1.13.12
	k8s.io/apimachinery v0.0.0-20191004074956-01f8b7d1121a // kubernetes-1.13.12
	k8s.io/client-go v0.0.0-20191004102537-eb5b9a8cfde7 // kubernetes-1.13.12
	k8s.io/klog v1.0.0 // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)
