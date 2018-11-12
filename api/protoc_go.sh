cd $GOPATH/src
protoc \
${GOPATH}/src/github.com/GoogleCloudPlatform/open-match/api/protobuf-spec/backend.proto \
${GOPATH}/src/github.com/GoogleCloudPlatform/open-match/api/protobuf-spec/frontend.proto \
${GOPATH}/src/github.com/GoogleCloudPlatform/open-match/api/protobuf-spec/mmlogic.proto \
${GOPATH}/src/github.com/GoogleCloudPlatform/open-match/api/protobuf-spec/messages.proto \
-I ${GOPATH}/src/github.com/GoogleCloudPlatform/open-match/ \
--go_out=plugins=grpc:$GOPATH/src
cd -
