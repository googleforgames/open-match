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
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/gobs/pretty"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	filename       = flag.String("file", "profiles/testprofile.json", "JSON file from which to read match properties")
	beCall         = flag.String("call", "ListMatches", "Open Match backend match request gRPC call to test")
	assignment     = flag.String("assignment", "example.server.dgs:12345", "Assignment to send to matched players")
	delAssignments = flag.Bool("rm", false, "Delete assignments. Leave off to be able to manually validate assignments in state storage")
	verbose        = flag.Bool("verbose", false, "Print out as much as possible")
)

func bytesToString(data []byte) string {
	return string(data[:])
}

func ppJSON(s string) {
	if *verbose {
		buf := new(bytes.Buffer)
		json.Indent(buf, []byte(s), "", "  ")
		log.Println(buf)
	}
	return
}

func main() {
	flag.Parse()
	log.Print("Parsing flags:")
	log.Printf(" [flags] Reading properties from file at %v", *filename)
	log.Printf(" [flags] Using OM Backend %v call", *beCall)
	log.Printf(" [flags] Assigning players to %v", *assignment)
	log.Printf(" [flags] Deleting assignments? %v", *delAssignments)
	if !(*beCall == "CreateMatch" || *beCall == "ListMatches") {
		log.Printf(" [flags] Unknown OM Backend call %v! Exiting...", *beCall)
		return
	}

	// Read the profile
	jsonFile, err := os.Open(*filename)
	if err != nil {
		log.Fatal("Failed to open file ", *filename)
	}
	defer jsonFile.Close()

	// parse json data and remove extra whitespace before sending to the backend.
	jsonData, _ := ioutil.ReadAll(jsonFile) // this reads as a byte array
	buffer := new(bytes.Buffer)             // convert byte array to buffer to send to json.Compact()
	if err := json.Compact(buffer, jsonData); err != nil {
		log.Println(err)
	}

	jsonProfile := buffer.String()
	pbProfile := &pb.MatchObject{}
	pbProfile.Properties = jsonProfile

	addr := "om-backendapi:50505"
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err.Error())
	}

	client := pb.NewBackendClient(conn)
	log.Println("Backend client connected to", addr)

	var profileName string
	if gjson.Get(jsonProfile, "name").Exists() {
		profileName = gjson.Get(jsonProfile, "name").String()
	} else {
		profileName = "testprofilename"
		log.Println("JSON Profile does not contain a name; using ", profileName)
	}

	pbProfile.Id = profileName
	pbProfile.Properties = jsonProfile

	mmfcfg := &pb.MmfConfig{Name: "profileName"}
	mmfcfg.Type = pb.MmfConfig_GRPC
	mmfcfg.Host = gjson.Get(jsonProfile, "hostname").String()
	mmfcfg.Port = int32(gjson.Get(jsonProfile, "port").Int())

	req := &pb.CreateMatchRequest{
		Match:  pbProfile,
		Mmfcfg: mmfcfg,
	}

	log.Println("Backend Request:")
	ppJSON(jsonProfile)
	pretty.PrettyPrint(mmfcfg)

	log.Printf("Establishing HTTPv2 stream...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	matchChan := make(chan *pb.MatchObject)
	doneChan := make(chan bool)
	go func() {
		// Watch for results and print as they come in.
		log.Println("Watching for match results...")
		for {
			select {
			case match := <-matchChan:
				if match.Error == "insufficient players" {
					log.Println("Waiting for a larger player pool...")
				}

				// Validate JSON before trying to  parse it
				if !gjson.Valid(string(match.Properties)) {
					log.Println(errors.New("invalid json"))
				}
				log.Println("Received match:")
				pretty.PrettyPrint(match)

				// Assign players in this match to our server
				log.Println("Assigning players to DGS at", *assignment)

				assign := &pb.Assignments{Rosters: match.Rosters, Assignment: *assignment}
				log.Printf("Waiting for matches...")
				_, err = client.CreateAssignments(context.Background(), &pb.CreateAssignmentsRequest{
					Assignment: assign,
				})

				if err != nil {
					log.Println(err)
				}
				log.Println("Success!")

				if *delAssignments {
					log.Println("deleting assignments")
					for _, a := range assign.Rosters {
						_, err = client.DeleteAssignments(context.Background(), &pb.DeleteAssignmentsRequest{Roster: a})
						if err != nil {
							log.Println(err)
						}
						log.Println("Success Deleting Assignments!")
					}
				} else {
					log.Println("Not deleting assignments [demo mode].")
				}
			}
			if *beCall == "CreateMatch" {
				// Got a result; done here.
				log.Println("Got single result from CreateMatch, exiting...")
				doneChan <- true
				return
			}
		}
	}()

	// Make the requested backend call: CreateMatch calls once, ListMatches continually calls.
	log.Printf("Attempting %v() call", *beCall)
	switch *beCall {
	case "CreateMatch":
		resp, err := client.CreateMatch(ctx, req)
		if err != nil {
			panic(err)
		}
		log.Printf("CreateMatch returned; processing match")

		matchChan <- resp.Match
		<-doneChan
	case "ListMatches":
		stream, err := client.ListMatches(ctx, &pb.ListMatchesRequest{
			Mmfcfg: req.Mmfcfg,
			Match:  req.Match,
		})
		if err != nil {
			log.Fatalf("Attempting to open stream for ListMatches(_) = _, %v", err)
		}
		for {
			log.Printf("Waiting for matches...")
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				stat, ok := status.FromError(err)
				if ok {
					log.Printf("Error reading stream for ListMatches() returned status: %s %s", stat.Code().String(), stat.Message())
				} else {
					log.Printf("Error reading stream for ListMatches() returned status: %s", err)
				}
				break
			}
			matchChan <- resp.Match
		}
	}
}
