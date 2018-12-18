FROM golang:1.10.3 as builder
WORKDIR /go/src/udp-client
COPY ./ ./
RUN go get -d -v 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o client . 

# final image
FROM alpine:3.8

RUN adduser -D client
COPY --from=builder /go/src/udp-client/client /home/client/client

COPY --from=builder /go/src/udp-client/city.percent /home/client/
COPY --from=builder /go/src/udp-client/europe-west* /home/client/

RUN chown -R client /home/client && \
    chmod o+x /home/client/client

USER client
WORKDIR /home/client
ENTRYPOINT /home/client/client