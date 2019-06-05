load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "6776d68ebb897625dead17ae510eac3d5f6342367327875210df44dbe2aeeb19",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.17.1/rules_go-0.17.1.tar.gz"],
)
load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

http_archive(
    name = "com_google_protobuf",
    sha256 = "d6618d117698132dadf0f830b762315807dc424ba36ab9183f1f436008a2fdb6",
    strip_prefix = "protobuf-3.6.1.2",
    urls = ["https://github.com/protocolbuffers/protobuf/archive/v3.6.1.2.zip"],
)

http_archive(
    name = "com_github_grpc-ecosystem_grpc-gateway",
    sha256 = "3b04ec65a50045cb04c7dc6f1644b4be5cd003b914774f52ac659ba5bc2d2a2c",
    strip_prefix = "grpc-gateway-1.9.0",
    urls = ["https://github.com/grpc-ecosystem/grpc-gateway/archive/v1.9.0.zip"],
)
