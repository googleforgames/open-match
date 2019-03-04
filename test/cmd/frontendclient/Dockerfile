FROM golang:1.10.3 as builder
WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match/test/cmd/frontendclient
COPY ./ ./
RUN go get -d -v 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o frontendclient . 

CMD ["./frontendclient"]
