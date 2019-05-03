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

module open-match.dev/open-match

go 1.12

require (
	github.com/GoogleCloudPlatform/open-match v0.5.0
	github.com/TV4/logrus-stackdriver-formatter v0.1.0
	github.com/alicebob/miniredis v2.5.0+incompatible
	github.com/cenkalti/backoff v2.1.1+incompatible
	github.com/fsnotify/fsnotify v1.4.7
	github.com/gobs/pretty v0.0.0-20180724170744-09732c25a95b
	github.com/gogo/protobuf v1.2.1
	github.com/golang/protobuf v1.3.1
	github.com/gomodule/redigo v1.7.0
	github.com/grpc-ecosystem/grpc-gateway v1.8.5
	github.com/opencensus-integrations/redigo v2.0.1+incompatible
	github.com/pkg/errors v0.8.0
	github.com/rafaeljusto/redigomock v0.0.0-20190202135759-257e089e14a1
	github.com/rs/xid v1.2.1
	github.com/sirupsen/logrus v1.4.1
	github.com/spf13/viper v1.3.2
	github.com/stretchr/testify v1.3.0
	github.com/tidwall/gjson v1.2.1
	go.opencensus.io v0.20.2
	golang.org/x/net v0.0.0-20190404232315-eb5bcb51f2a3
	google.golang.org/genproto v0.0.0-20190404172233-64821d5d2107
	google.golang.org/grpc v1.20.0
)
