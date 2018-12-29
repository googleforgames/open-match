# This will be unnecessary once pb/ package become 'go-get'-able
FROM ihrankouski/open-match-base-build:dev as builder

WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match/examples/director/cmd/director
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .

FROM alpine:3.8

RUN adduser -D director
COPY --from=builder \
    --chown=director \
    /go/src/github.com/GoogleCloudPlatform/open-match/examples/director/cmd/director/director \
    /home/director/director

USER director
WORKDIR /home/director
ENTRYPOINT ["/home/director/director"]
