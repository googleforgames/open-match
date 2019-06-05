proto_library(
    visibility = ["//visibility:public"],
    name = "backend_proto",
    srcs = ["backend.proto"],
    deps = [
        "//api:messages_proto",
        "@com_github_grpc-ecosystem_grpc-gateway//protoc-gen-swagger/options:options_proto",
        #"@go_googleapis//google/api:annotations_proto",
        #"@com_google_protobuf//:struct_proto"
    ],
)

proto_library(
    visibility = ["//visibility:public"],
    name = "evaluator_proto",
    srcs = ["evaluator.proto"],
    deps = [],
)

proto_library(
    visibility = ["//visibility:public"],
    name = "frontend_proto",
    srcs = ["frontend.proto"],
    deps = [],
)

proto_library(
    visibility = ["//visibility:public"],
    name = "matchfunction_proto",
    srcs = ["matchfunction.proto"],
    deps = [],
)

proto_library(
    visibility = ["//visibility:public"],
    name = "messages_proto",
    srcs = ["messages.proto"],
    deps = [
        "@com_google_protobuf//:struct_proto"
    ],
)

proto_library(
    visibility = ["//visibility:public"],
    name = "mmlogic_proto",
    srcs = ["mmlogic.proto"],
    deps = [],
)
