# Golang application builder steps
FROM gcr.io/matchmaker-dev-201405/openmatch-base:dev as builder
WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match
COPY cmd/mmlogicapi cmd/mmlogicapi
WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match/cmd/mmlogicapi
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .

#FROM scratch
#COPY --from=builder /go/src/github.com/GoogleCloudPlatform/open-match/cmd/frontendapi/frontendapi . 
ENTRYPOINT ["./mmlogicapi"]
