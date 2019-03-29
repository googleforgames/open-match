# Open Match APIs

This directory contains the API specification files for Open Match. API documenation will be produced in a future version, although the protobuf files offer a concise description of the API calls available, along with arguments and return messages.

* [Protobuf .proto files for all APIs](./protobuf-spec/)

References:

* [gRPC](https://grpc.io/)
* [Language Guide (proto3)](https://developers.google.com/protocol-buffers/docs/proto3)

If you want to regenerate the golang gRPC modules (for local Open Match core component development, for example), the `protoc_go.sh` file in this directory may be of use to you!
