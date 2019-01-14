# Golang application builder steps
FROM gcr.io/open-match-222408/openmatch-base:dev as builder
WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match/cmd/backendapi
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .

FROM alpine:3.8
WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match/cmd/backendapi
COPY --from=builder /go/src/github.com/GoogleCloudPlatform/open-match/config/matchmaker_config.yaml ../../config/
RUN ln -s ../../config/matchmaker_config.yaml
COPY --from=builder /go/src/github.com/GoogleCloudPlatform/open-match/cmd/backendapi/backendapi .

ENTRYPOINT ["./backendapi"]
