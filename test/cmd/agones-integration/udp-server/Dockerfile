FROM golang:1.10.3 as builder
WORKDIR /go/src/udp-server
COPY ./ ./
RUN go get -d -v 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server . 

# final image
FROM alpine:3.8

RUN adduser -D server
COPY --from=builder /go/src/udp-server/server /home/server/server
RUN chown -R server /home/server && \
    chmod o+x /home/server/server

USER server
ENTRYPOINT /home/server/server