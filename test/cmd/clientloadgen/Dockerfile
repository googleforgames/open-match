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

FROM open-match-base-build as builder

WORKDIR /go/src/github.com/googleforgames/open-match/test/cmd/clientloadgen/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o clientloadgen . 

FROM gcr.io/distroless/static  
COPY --from=builder /go/src/github.com/googleforgames/open-match/test/cmd/clientloadgen/clientloadgen .
COPY --from=builder /go/src/github.com/googleforgames/open-match/test/cmd/clientloadgen/*.percent ./
COPY --from=builder /go/src/github.com/googleforgames/open-match/test/cmd/clientloadgen/*.ping ./

ENTRYPOINT ["/clientloadgen"]
