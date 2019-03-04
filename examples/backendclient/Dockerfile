#FROM golang:1.10.3 as builder
FROM gcr.io/open-match-public-images/openmatch-base:dev as builder
WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match/examples/backendclient
COPY ./ ./
RUN go get -d -v 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o backendclient . 

CMD ["./backendclient"]
