// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.19.4
// source: api/evaluator.proto

package pb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Evaluator_Evaluate_FullMethodName = "/openmatch.Evaluator/Evaluate"
)

// EvaluatorClient is the client API for Evaluator service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type EvaluatorClient interface {
	// Evaluate evaluates a list of proposed matches based on quality, collision status, and etc, then shortlist the matches and returns the final results.
	Evaluate(ctx context.Context, opts ...grpc.CallOption) (Evaluator_EvaluateClient, error)
}

type evaluatorClient struct {
	cc grpc.ClientConnInterface
}

func NewEvaluatorClient(cc grpc.ClientConnInterface) EvaluatorClient {
	return &evaluatorClient{cc}
}

func (c *evaluatorClient) Evaluate(ctx context.Context, opts ...grpc.CallOption) (Evaluator_EvaluateClient, error) {
	stream, err := c.cc.NewStream(ctx, &Evaluator_ServiceDesc.Streams[0], Evaluator_Evaluate_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &evaluatorEvaluateClient{stream}
	return x, nil
}

type Evaluator_EvaluateClient interface {
	Send(*EvaluateRequest) error
	Recv() (*EvaluateResponse, error)
	grpc.ClientStream
}

type evaluatorEvaluateClient struct {
	grpc.ClientStream
}

func (x *evaluatorEvaluateClient) Send(m *EvaluateRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *evaluatorEvaluateClient) Recv() (*EvaluateResponse, error) {
	m := new(EvaluateResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// EvaluatorServer is the server API for Evaluator service.
// All implementations should embed UnimplementedEvaluatorServer
// for forward compatibility
type EvaluatorServer interface {
	// Evaluate evaluates a list of proposed matches based on quality, collision status, and etc, then shortlist the matches and returns the final results.
	Evaluate(Evaluator_EvaluateServer) error
}

// UnimplementedEvaluatorServer should be embedded to have forward compatible implementations.
type UnimplementedEvaluatorServer struct {
}

func (UnimplementedEvaluatorServer) Evaluate(Evaluator_EvaluateServer) error {
	return status.Errorf(codes.Unimplemented, "method Evaluate not implemented")
}

// UnsafeEvaluatorServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to EvaluatorServer will
// result in compilation errors.
type UnsafeEvaluatorServer interface {
	mustEmbedUnimplementedEvaluatorServer()
}

func RegisterEvaluatorServer(s grpc.ServiceRegistrar, srv EvaluatorServer) {
	s.RegisterService(&Evaluator_ServiceDesc, srv)
}

func _Evaluator_Evaluate_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(EvaluatorServer).Evaluate(&evaluatorEvaluateServer{stream})
}

type Evaluator_EvaluateServer interface {
	Send(*EvaluateResponse) error
	Recv() (*EvaluateRequest, error)
	grpc.ServerStream
}

type evaluatorEvaluateServer struct {
	grpc.ServerStream
}

func (x *evaluatorEvaluateServer) Send(m *EvaluateResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *evaluatorEvaluateServer) Recv() (*EvaluateRequest, error) {
	m := new(EvaluateRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Evaluator_ServiceDesc is the grpc.ServiceDesc for Evaluator service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Evaluator_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "openmatch.Evaluator",
	HandlerType: (*EvaluatorServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Evaluate",
			Handler:       _Evaluator_Evaluate_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "api/evaluator.proto",
}
