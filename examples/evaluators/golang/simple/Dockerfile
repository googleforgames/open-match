FROM open-match-base-build as builder

WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match/examples/evaluators/golang/simple/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .

FROM gcr.io/distroless/static  
COPY --from=builder /go/src/github.com/GoogleCloudPlatform/open-match/examples/evaluators/golang/simple/simple .
# TODO: Remove embedding the config once loading configuration is normalized and this can respect the ConfigMap.
COPY --from=builder /go/src/github.com/GoogleCloudPlatform/open-match/config/matchmaker_config.yaml ./config/

ENTRYPOINT ["/simple"]
