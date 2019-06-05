load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")


# Protobuf Things
http_archive(
    name = "com_google_protobuf",
    sha256 = "d6618d117698132dadf0f830b762315807dc424ba36ab9183f1f436008a2fdb6",
    strip_prefix = "protobuf-3.6.1.2",
    urls = ["https://github.com/protocolbuffers/protobuf/archive/v3.6.1.2.zip"],
)

# Go Things
http_archive(
    name = "io_bazel_rules_go",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.18.5/rules_go-0.18.5.tar.gz"],
    sha256 = "a82a352bffae6bee4e95f68a8d80a70e87f42c4741e6a448bec11998fcc82329",
)
load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains()

# GRPC Gateway
http_archive(
    name = "com_github_grpc_ecosystem_grpc_gateway",
    sha256 = "2ab54450b70b526a1d26ebd2ff02c9a2e8cbdaae3ed0a1d550230dbe098fc6cb",
    strip_prefix = "grpc-gateway-21f5e5895efe7cad510ff5b9a77659c77b9a8856",
    urls = ["https://github.com/cgrinker/grpc-gateway/archive/21f5e5895efe7cad510ff5b9a77659c77b9a8856.zip"],
)