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


FROM python:3.5 AS build-env
# Install the required dependencies via pip
RUN pip install locustio
# Add the external tasks directory into the container image
ADD test/stress test/stress

FROM gcr.io/distroless/python3
# Copy Locust binaries
COPY --from=build-env /usr/local/lib/python3.5/site-packages/ /usr/local/lib/python3.5/site-packages/
COPY --from=build-env /usr/local/bin/locust /test/stress/locust
COPY --from=build-env test/stress /test/stress
ENV PYTHONPATH=/usr/local/lib/python3.5/site-packages
# Expose the required Locust ports
EXPOSE 5557 5558 8089
WORKDIR /test/stress
