# Open Match API

Open Match API is exposed via [gRPC](https://grpc.io/) and HTTP REST with [Swagger](https://swagger.io/tools/swagger-codegen/).

gRPC has first-class support for [many languages](https://grpc.io/docs/) and provides the most performance. It is a RPC protocol built on top of HTTP/2 and provides TLS for secure transport.

For HTTP/HTTPS Open Match uses a gRPC proxy to serve the API. Since HTTP does not provide a structure for request/responses we use Swagger to provide a schema. You can view the Swagger docs for each service in this directory's `*.swagger.json` files. In addition each server will host it's swagger doc via `GET /swagger.json` if you want to dynamically load them at runtime.

Lastly, Open Match supports insecure and TLS mode for serving the API. It's strongly preferred to use TLS mode in production but insecure mode can be used for test and local development. To help with certificates management see `tools/certgen` to create self-signed certificates.

# Open Match API Development Guide

Open Match proto comments follow the same format as [this file](https://github.com/tensorflow/tensorflow/blob/master/tensorflow/core/example/example.proto)

If you plan to change the proto definitions, please update the comments and run `make api/api.md` to reflect the latest changes in open-match-docs.