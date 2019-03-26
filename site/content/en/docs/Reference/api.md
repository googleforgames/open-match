---
title: "API"
linkTitle: "API"
date: 2017-01-05
description: >
  A short lead descripton about this content page. It can be **bold** or _italic_ and can be split over multiple paragraphs.
---

# Open Match APIs

This directory contains the API specification files for Open Match. API documenation will be produced in a future version, although the protobuf files offer a concise description of the API calls available, along with arguments and return messages.

* [Protobuf .proto files for all APIs](./protobuf-spec/)

These proto files are copied to the container image during `docker build` for the Open Match core components.  The `Dockerfiles` handle the compilation for you transparently, and copy the resulting `SPEC.pb.go` files to the appropriate place in your final container image.

References:

* [gRPC](https://grpc.io/)
* [Language Guide (proto3)](https://developers.google.com/protocol-buffers/docs/proto3)

Manual gRPC compilation commmand, from the directory containing the proto:
```protoc -I . ./<filename>.proto --go_out=plugins=grpc:.```

# REST compatibility 
Follow the guidelines at https://cloud.google.com/endpoints/docs/grpc/transcoding
to keep the gRPC service definitions friendly to REST transcoding. An excerpt:

"Transcoding involves mapping HTTP/JSON requests and their parameters to gRPC
methods and their parameters and return types (we'll look at exactly how you
do this in the following sections). Because of this, while it's possible to
map an HTTP/JSON request to any arbitrary API method, it's simplest and most
intuitive to do so if the gRPC API itself is structured in a
resource-oriented way, just like a traditional HTTP REST API. In other
words, the API service should be designed so that it uses a small number of
standard methods (corresponding to HTTP verbs like GET, PUT, and so on) that
operate on the service's resources (and collections of resources, which are
themselves a type of resource). 
These standard methods are List, Get, Create, Update, and Delete."

It is for these reasons we don't have gRPC calls that support bi-directional streaming in Open Match.