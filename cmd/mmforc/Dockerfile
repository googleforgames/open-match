# Golang application builder steps
FROM gcr.io/matchmaker-dev-201405/openmatch-base:dev as builder

# Necessary to get a specific version of the golang k8s client
RUN go get github.com/tools/godep
RUN go get k8s.io/client-go/...
WORKDIR /go/src/k8s.io/client-go
RUN git checkout v7.0.0
RUN godep restore ./...
RUN rm -rf vendor/
RUN rm -rf /go/src/github.com/golang/protobuf/

WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match/
COPY cmd/mmforc cmd/mmforc
WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match/cmd/mmforc/
RUN go get -d -v 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .

# Uncomment to build production images (removes all troubleshooting tools)
#FROM scratch
#COPY --from=builder /go/src/github.com/GoogleCloudPlatform/open-match/cmd/mmforc/mmforc .
CMD ["./mmforc"]
