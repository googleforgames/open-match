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

FROM golang:alpine as go
WORKDIR /app
ENV GO111MODULE=on

COPY . .
RUN go mod edit -replace open-match.dev/open-match@v0.0.0-dev=open-match.dev/open-match@v1.7.0
RUN go mod tidy
RUN go build -o frontend .

CMD ["/app/frontend"]
