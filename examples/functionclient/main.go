/*
Stubbed backend api client. This should be run within a k8s cluster, and
assumes that the backend api is up and can be accessed through a k8s service
named om-backendapi

Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net"
	"os"

	backend "github.com/GoogleCloudPlatform/open-match/internal/pb"
	"google.golang.org/grpc"
)

func bytesToString(data []byte) string {
	return string(data[:])
}

func ppJSON(s string) {
	buf := new(bytes.Buffer)
	json.Indent(buf, []byte(s), "", "  ")
	log.Println(buf)
	return
}

func main() {

	// Connect gRPC client
	ip, err := net.LookupHost("om-function")
	if err != nil {
		panic(err)
	}

	conn, err := grpc.Dial(ip[0]+":50502", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err.Error())
	}
	client := backend.NewFunctionClient(conn)
	log.Println("API client connected to", ip[0]+":50502")

	/*
		profileName := "test-dm-usc1f"
		_ = profileName
		if gjson.Get(jsonProfile, "name").Exists() {
			profileName = gjson.Get(jsonProfile, "name").String()
		}

		pbProfile.Id = profileName
		pbProfile.Properties = jsonProfile
	*/
	args := backend.Arguments{
		Request: &backend.Request{
			ProfileId:  os.Args[1],
			ProposalId: os.Args[2],
		},
		Matchobject: &backend.MatchObject{},
	}

	log.Printf("Establishing HTTPv2 stream...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	match, err := client.Run(ctx, &args)
	log.Printf("results: %v, %v\n", match, err)

	/*
	 */
}
