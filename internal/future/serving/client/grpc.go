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

package client

import (
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/future/serving"
)

// TODO: provides TLS client support

// InsecureGRPCFromConfig creates a gRPC client connection without TLS support from a configuration.
func InsecureGRPCFromConfig(cfg config.View, prefix string) (*grpc.ClientConn, error) {
	hostname := cfg.GetString(prefix + ".hostname")
	portNumber := cfg.GetInt(prefix + ".grpcport")
	return GRPCFromParams(&Params{
		Hostname: hostname,
		Port:     portNumber,
	})
}

// GRPCFromParams creates a gRPC client connection from the parameters.
func GRPCFromParams(params *Params) (*grpc.ClientConn, error) {
	address := fmt.Sprintf("%s:%d", params.Hostname, params.Port)
	if params.usingTLS() {
		creds, err := serving.ClientCredentialsFromFileData(params.PublicCertificate, address)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return grpc.Dial(address, grpc.WithTransportCredentials(creds))
	}
	return grpc.Dial(address, grpc.WithInsecure())
}
