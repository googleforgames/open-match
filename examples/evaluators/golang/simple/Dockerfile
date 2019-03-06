# Golang application builder steps
FROM golang:1.11.5 as builder
WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match
# Get only the necessary files from the repo.
# https://stackoverflow.com/a/39317180/3113674
# TODO: Set these to the final branch name after RC approval.
RUN svn export https://github.com/joeholley/open-match/branches/gdcdemo/internal
RUN svn export https://github.com/joeholley/open-match/branches/gdcdemo/config
WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match/examples/evaluators/golang/simple/
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux go build -o evaluator -a -installsuffix cgo . 

#FROM gcr.io/distroless/static  
FROM debian
COPY --from=builder /go/src/github.com/GoogleCloudPlatform/open-match/examples/evaluators/golang/simple/evaluator /.  
ENTRYPOINT ["./evaluator"]
