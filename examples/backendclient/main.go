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
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"

	backend "github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
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

	// Read the profile
	filename := "profiles/testprofile.json"
	/*
		if len(os.Args) > 1 {
			filename = os.Args[1]
		}
		log.Println("Reading profile from ", filename)
	*/
	jsonFile, err := os.Open(filename)
	if err != nil {
		panic("Failed to open file specified at command line.  Did you forget to specify one?")
	}
	defer jsonFile.Close()

	// parse json data and remove extra whitespace before sending to the backend.
	jsonData, _ := ioutil.ReadAll(jsonFile) // this reads as a byte array
	buffer := new(bytes.Buffer)             // convert byte array to buffer to send to json.Compact()
	if err := json.Compact(buffer, jsonData); err != nil {
		log.Println(err)
	}

	jsonProfile := buffer.String()
	pbProfile := &backend.MatchObject{}
	/*
		err = jsonpb.UnmarshalString(jsonProfile, pbProfile)
		if err != nil {
			log.Println(err)
		}
	*/
	pbProfile.Properties = jsonProfile

	log.Println("Requesting matches that fit profile:")
	ppJSON(jsonProfile)
	//jsonProfile := bytesToString(jsonData)

	// Connect gRPC client
	ip, err := net.LookupHost("om-backendapi")
	if err != nil {
		panic(err)
	}

	conn, err := grpc.Dial(ip[0]+":50505", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err.Error())
	}
	client := backend.NewBackendClient(conn)
	log.Println("API client connected to", ip[0]+":50505")

	profileName := "test-dm-usc1f"
	_ = profileName
	if gjson.Get(jsonProfile, "name").Exists() {
		profileName = gjson.Get(jsonProfile, "name").String()
	}

	pbProfile.Id = profileName
	pbProfile.Properties = jsonProfile

	log.Printf("Establishing HTTPv2 stream...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	//match, err := client.CreateMatch(ctx, pbProfile)

	for {
		log.Println("Attempting to send ListMatches call")
		stream, err := client.ListMatches(ctx, pbProfile)
		if err != nil {
			log.Fatalf("Attempting to open stream for ListMatches(_) = _, %v", err)
		}
		//for i := 0; i < 2; i++ {
		for {
			log.Printf("Waiting for matches...")
			match, err := stream.Recv()
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

			if match.Properties == "{error: insufficient_players}" {
				log.Println("Waiting for a larger player pool...")
				//break
			}

			// Validate JSON before trying to  parse it
			if !gjson.Valid(string(match.Properties)) {
				log.Println(errors.New("invalid json"))
			}
			log.Println("Received match", match.Id)
			ppJSON(match.Properties)
			//fmt.Println(match)  // Debug

			// Assign players in this match to our server
			connstring := "example.com:12345"
			if len(os.Args) >= 2 {
				connstring = os.Args[1]
				log.Printf("Player assignment '%v' specified at commandline", connstring)
			}
			log.Println("Assigning players to DGS at", connstring)

			assign := &backend.Assignments{Rosters: match.Rosters, Assignment: connstring}
			log.Printf("Waiting for matches...")
			_, err = client.CreateAssignments(context.Background(), assign)
			if err != nil {
				stat, ok := status.FromError(err)
				if ok {
					log.Printf("Error reading stream for ListMatches() returned status: %s %s", stat.Code().String(), stat.Message())
				} else {
					log.Printf("Error reading stream for ListMatches() returned status: %s", err)
				}
				break
			}
			log.Println("Success!  Not deleting assignments [demo mode].")

		}

		//log.Println("deleting assignments")
		//playerstr = strings.Join(players[0:len(players)/2], " ")
		//roster.PlayerIds = playerstr
		//_, err = client.DeleteAssignments(context.Background(), roster)
	}
}
