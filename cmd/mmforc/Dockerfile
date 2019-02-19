# Golang application builder steps
FROM golang:1.11.5 as builder 

# Necessary to get a specific version of the golang k8s client
RUN go get github.com/tools/godep
RUN go get k8s.io/client-go/...
WORKDIR /go/src/k8s.io/client-go
RUN git checkout v7.0.0
RUN godep restore ./...
RUN rm -rf vendor/
RUN rm -rf /go/src/github.com/golang/protobuf/

# Get only the necessary files from the repo.
# https://stackoverflow.com/a/39317180/3113674
# TODO: Set these to the final branch name after RC approval.
WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match
RUN svn export https://github.com/GoogleCloudPlatform/open-match/branches/040wip/internal
RUN svn export https://github.com/GoogleCloudPlatform/open-match/branches/040wip/config

WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match/cmd/mmforc/
COPY . .
RUN go get -d -v 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .

FROM gcr.io/distroless/static
COPY --from=builder /go/src/github.com/GoogleCloudPlatform/open-match/cmd/mmforc/mmforc .

CMD ["/mmforc"]
