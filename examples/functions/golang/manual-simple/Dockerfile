# Golang application builder steps
FROM gcr.io/open-match-public-images/openmatch-base:dev as builder
WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match/examples/functions/golang/manual-simple
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mmf .

FROM gcr.io/distroless/static
COPY --from=builder /go/src/github.com/GoogleCloudPlatform/open-match/examples/functions/golang/manual-simple/mmf .
CMD ["/mmf"]
