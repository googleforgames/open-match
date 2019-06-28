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

WORKDIR /go/src/open-match.dev/open-match/cmd/swaggerui/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .
COPY api/*.json /go/src/open-match.dev/open-match/third_party/swaggerui/api/

# Since we copy the swagger docs to the container point to them so they are served locally.
# This is important because if there are local changes we want those reflecting in the container.
RUN sed -i 's|https://open-match.dev/api/v.*/|/api/|g' /go/src/open-match.dev/open-match/third_party/swaggerui/config.json

FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=builder --chown=nonroot /go/src/open-match.dev/open-match/cmd/swaggerui/swaggerui /app/
COPY --from=builder --chown=nonroot /go/src/open-match.dev/open-match/third_party/swaggerui/ /app/static
ENTRYPOINT ["/app/swaggerui"]

# Docker Image Arguments
ARG BUILD_DATE
ARG VCS_REF
ARG BUILD_VERSION
ARG IMAGE_TITLE="Open Match Swagger UI"

# Standardized Docker Image Labels
# https://github.com/opencontainers/image-spec/blob/master/annotations.md
LABEL \
    org.opencontainers.image.created="${BUILD_TIME}" \
    org.opencontainers.image.authors="Google LLC <open-match-discuss@googlegroups.com>" \
    org.opencontainers.image.url="https://open-match.dev/" \
    org.opencontainers.image.documentation="https://open-match.dev/site/docs/" \
    org.opencontainers.image.source="https://github.com/GoogleCloudPlatform/open-match" \
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
    org.label-schema.vcs-url="https://github.com/GoogleCloudPlatform/open-match" \
    org.label-schema.version=$BUILD_VERSION \
    org.label-schema.vcs-ref=$VCS_REF \
    org.label-schema.vendor="Google LLC" \
    org.label-schema.name="${IMAGE_TITLE}" \
    org.label-schema.description="Flexible, extensible, and scalable video game matchmaking." \
    org.label-schema.usage="https://open-match.dev/site/docs/"
