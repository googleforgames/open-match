FROM open-match-base-build as builder

WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match/test/cmd/frontendclient/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o frontendclient . 

FROM gcr.io/distroless/static  
COPY --from=builder /go/src/github.com/GoogleCloudPlatform/open-match/test/cmd/frontendclient/frontendclient .
COPY --from=builder /go/src/github.com/GoogleCloudPlatform/open-match/test/cmd/frontendclient/*.percent ./
COPY --from=builder /go/src/github.com/GoogleCloudPlatform/open-match/test/cmd/frontendclient/*.ping ./

ENTRYPOINT ["/frontendclient"]
