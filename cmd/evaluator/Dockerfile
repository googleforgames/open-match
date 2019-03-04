# Golang application builder steps
FROM golang:1.11.5 as builder 

# Get only the necessary files from the repo.
# https://stackoverflow.com/a/39317180/3113674
# TODO: Set these to the final branch name after RC approval.
WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match
RUN svn export https://github.com/joeholley/open-match/branches/gdcdemo/internal
RUN svn export https://github.com/joeholley/open-match/branches/gdcdemo/config

WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match/cmd/evaluator/
COPY . .
RUN go get -d -v 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .

#FROM gcr.io/distroless/static
FROM debian
COPY --from=builder /go/src/github.com/GoogleCloudPlatform/open-match/cmd/evaluator/evaluator .

CMD ["./evaluator"]
