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

WORKDIR /go/src/open-match.dev/open-match/cmd/minimatch/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .

FROM gcr.io/distroless/static:nonroot
WORKDIR /app/
COPY --from=builder --chown=nonroot /go/src/open-match.dev/open-match/cmd/minimatch/minimatch /app/

ENTRYPOINT ["/app/minimatch"]

# Docker Image Arguments
ARG BUILD_DATE
ARG VCS_REF
ARG BUILD_VERSION
ARG IMAGE_TITLE="Mini Match"

# Standardized Docker Image Labels
# https://github.com/opencontainers/image-spec/blob/master/annotations.md
LABEL \
    org.opencontainers.image.created="${BUILD_TIME}" \
    org.opencontainers.image.authors="Google LLC <open-match-discuss@googlegroups.com>" \
    org.opencontainers.image.url="https://open-match.dev/" \
    org.opencontainers.image.documentation="https://open-match.dev/site/docs/" \
    org.opencontainers.image.source="https://github.com/googleforgames/open-match" \
    org.opencontainers.image.version="${BUILD_VERSION}" \
    org.opencontainers.image.revision="1" \
    org.opencontainers.image.vendor="Google LLC" \
    org.opencontainers.image.licenses="Apache-2.0" \
    org.opencontainers.image.ref.name="" \
    org.opencontainers.image.title="${IMAGE_TITLE}" \
    org.opencontainers.image.description="Flexible, extensible, and scalable video game matchmaking." \
    org.label-schema.schema-version="1.0" \
    org.label-schema.build-date=$BUILD_DATE \
    org.label-schema.url="http://open-match.dev/" \
    org.label-schema.vcs-url="https://github.com/googleforgames/open-match" \
    org.label-schema.version=$BUILD_VERSION \
    org.label-schema.vcs-ref=$VCS_REF \
    org.label-schema.vendor="Google LLC" \
    org.label-schema.name="${IMAGE_TITLE}" \
    org.label-schema.description="Flexible, extensible, and scalable video game matchmaking." \
    org.label-schema.usage="https://open-match.dev/site/docs/"
