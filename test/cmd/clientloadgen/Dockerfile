FROM open-match-base-build as builder

WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match/test/cmd/clientloadgen/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o clientloadgen . 

FROM gcr.io/distroless/static  
COPY --from=builder /go/src/github.com/GoogleCloudPlatform/open-match/test/cmd/clientloadgen/clientloadgen .

ENTRYPOINT ["/clientloadgen"]
