FROM golang:1.11.5 as builder
WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match/internal
# Get only the necessary files from the repo.
# https://stackoverflow.com/a/39317180/3113674
# TODO: Set these to the final branch name after RC approval.
RUN svn export https://github.com/joeholley/open-match/branches/gdcdemo/internal/pb
WORKDIR /go/src/github.com/GoogleCloudPlatform/open-match/examples/matchmaker
COPY ./ ./
RUN go get -d -v 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o matchmaker . 

CMD ["./matchmaker"]
