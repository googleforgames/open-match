/*
Stubbed pb api client. This should be run within a k8s cluster, and
assumes that the pb api is up and can be accessed through a k8s service
named om-pbapi

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
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/open-match/examples/matchmaker/mo"
	"github.com/gobs/pretty"

	"github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc"
)

func bytesToString(data []byte) string {
	return string(data[:])
}

func ppJSON(s string) {
	if verbose {
		buf := new(bytes.Buffer)
		json.Indent(buf, []byte(s), "", "  ")
		log.Println(buf)
	}
	return
}

var (
	profileName    string
	filename       string
	mmfType        string
	beHost         string
	bePort         string
	beCall         string
	assignment     string
	delAssignments bool
	verbose        bool
	resultsOnly    bool
	summary        bool

	concurrency int
	wg          sync.WaitGroup
)

func main() {

	// TODO: make flags if prove to be useful
	printJSON := false

	// Parse flags
	flag.IntVar(&concurrency, "concurrency", 20, "[NYI] Max number of backend API calls to run concurrently")
	flag.StringVar(&filename, "file", "profiles/testprofile.json", "JSON file from which to read match properties")
	flag.StringVar(&mmfType, "type", "grpc", "MMF type")
	flag.StringVar(&beCall, "call", "ListMatches", "Open Match backend match request gRPC call to test")
	flag.StringVar(&beHost, "host", "om-backendapi", "Open Match backend hostname")
	flag.StringVar(&bePort, "port", "50505", "Open Match backend port")
	flag.StringVar(&assignment, "assignment", "", "Assignment to send to matched players, set to empty to skip assigning")
	flag.BoolVar(&delAssignments, "rm", false, "Delete assignments. Leave off to be able to manually validate assignments in state storage")
	flag.BoolVar(&verbose, "verbose", false, "Print out as much as possible")
	flag.BoolVar(&resultsOnly, "resultsonly", false, "Print out only results")
	flag.BoolVar(&resultsOnly, "summary", false, "Print out only final summary")
	flag.Parse()

	if !summary && !resultsOnly {
		log.Print("Parsing flags:")
		log.Printf(" [flags] Reading properties from file at %v", filename)
		log.Printf(" [flags] Connecting to OM Backend at %v:%v", beHost, bePort)
		if !(beCall == "CreateMatch" || beCall == "ListMatches") {
			log.Printf(" [flags] Unknown OM Backend call %v! Exiting...", beCall)
			return
		}
		log.Printf(" [flags] Max concurrent OM Backend calls %v", concurrency)
		log.Printf(" [flags] Using OM Backend %v call", beCall)
		log.Printf(" [flags] Calling MMF via %v", mmfType)
		log.Printf(" [flags] Assigning players to %v", assignment)
		log.Printf(" [flags] Deleting assignments? %v", delAssignments)

	}
	// Read the profile
	jsonFile, err := os.Open(filename)
	if err != nil {
		log.Fatal("Failed to open file ", filename)
	}
	defer jsonFile.Close()

	// parse json data and remove extra whitespace before sending to the pb.
	jsonData, _ := ioutil.ReadAll(jsonFile) // this reads as a byte array
	buffer := new(bytes.Buffer)             // convert byte array to buffer to send to json.Compact()
	if err := json.Compact(buffer, jsonData); err != nil {
		log.Println(err)
	}

	jsonProfile := buffer.String()
	pbProfile := &pb.MatchObject{}
	pbProfile.Properties = jsonProfile

	// Connect gRPC client
	ip, err := net.LookupHost(beHost)
	if err != nil {
		panic(err)
	}
	conn, err := grpc.Dial(ip[0]+":"+bePort, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err.Error())
	}
	client := pb.NewBackendClient(conn)
	if !summary && !resultsOnly {
		log.Println("Backend client connected to", beHost+":"+bePort)
	}

	if gjson.Get(jsonProfile, "name").Exists() {
		profileName = gjson.Get(jsonProfile, "name").String()
	} else {
		profileName = "testprofilename"
		log.Println("JSON Profile does not contain a name; using ", profileName)
	}

	pbProfile.Id = profileName
	pbProfile.Properties = jsonProfile

	// Generate Job Spec for different ways of running MMFs
	mmfspec := &pb.MmfSpec{Name: "profileName"}
	switch mmfType {
	case "grpc":
		mmfspec.Type = pb.MmfSpec_GRPC
		mmfspec.Host = gjson.Get(jsonProfile, "hostname").String()
		mmfspec.Port = int32(gjson.Get(jsonProfile, "port").Int())
	case "job":
		mmfspec.Type = pb.MmfSpec_K8SJOB
	}
	req := pb.CreateMatchRequest{Mmfspec: mmfspec}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	failcount := 0
	okcount := 0

	// Analyze match objects as they come in.
	matchChan := make(chan *pb.MatchObject)
	go func() {
		for match := range matchChan {
			errOutput := ""

			if match.Error != "" {
				errOutput = fmt.Sprintf("| %v", match.Error)
				failcount++
			} else {
				okcount++
			}
			var poolOutput string
			poolOutput = ""
			if len(match.Pools) > 0 &&
				match.Pools[0].Stats != nil &&
				match.Pools[0].Stats.Count != 0 {
				poolOutput = fmt.Sprintf("%6v", match.Pools[0].Stats.Count)
				if match.Pools[0].Stats.Elapsed != 0 {
					poolOutput = fmt.Sprintf("%v | Elapsed %05.2f", poolOutput, match.Pools[0].Stats.Elapsed)
				}
			} else {
				poolOutput = fmt.Sprintf("       | Elapsed  0.0 ")
			}

			// Dump some info to screen
			if !summary {
				log.Printf("%31v | pools %6v %v",
					strings.Split(match.Id, ".")[1],
					poolOutput,
					errOutput,
				)
			}

			// Validate JSON before trying to  parse it
			if printJSON {
				if !gjson.Valid(string(match.Properties)) {
					log.Println(errors.New("invalid json"))
				}
				pretty.PrettyPrint(match.Properties)
			}

			// Assign players in this match to our server
			if assignment != "" {
				if verbose {
					log.Println("Assigning players to DGS at", assignment)
				}

				assign := &pb.Assignments{Rosters: match.Rosters, Assignment: assignment}
				_, err = client.CreateAssignments(context.Background(), assign)
				if err != nil {
					log.Println(err)
				}

				if delAssignments {
					log.Println("deleting assignments")
					for _, a := range assign.Rosters {
						_, err = client.DeleteAssignments(context.Background(), a)
					}
				}
			}

			// Mark processing for this pb call as done
			wg.Done()
		}
	}()

	// Make the requested pb call: CreateMatch calls once, ListMatches continually calls.
	start := time.Now()
	switch beCall {
	case "CreateMatch":
		// Get all combinations of profiles
		moChan := make(chan *pb.MatchObject)
		//go mo.GenerateMatchObjects(moChan)
		go mo.ProcedurallyGenerateMatchObjects(concurrency, moChan)

		for {
			//for requestMO := range moChan {
			select {
			case requestMO, ok := <-moChan:
				//go func(requestMO *pb.MatchObject) {
				if !ok {
					moChan = nil
				} else {
					go func() {
						// waitgroup
						wg.Add(1)
						//defer wg.Done()
						requestMO.Properties = pbProfile.Properties
						req := pb.CreateMatchRequest{
							Matchobject: requestMO,
							Mmfspec:     mmfspec,
						}
						if !resultsOnly && !summary {
							log.Printf(" Attempting %22v teams: %v * 8",
								req.Matchobject.Id,
								len(req.Matchobject.Rosters))
						}

						// Call pb to fill this matchobject
						match, err := client.CreateMatch(ctx, &req)
						if err != nil {
							wg.Done()
							if !resultsOnly && !summary {
								log.Printf("  Failed match: %42v | %v", req.Matchobject.Id, err)
							}
							failcount++
							return
						}

						// Got matchobject, send to the printer
						matchChan <- match
					}()
				}
			}
			if moChan == nil {
				break
			}
		}
		wg.Wait()
		close(matchChan)
		log.Printf("Total Runtime: %v", time.Since(start))
		log.Printf("Number of Failures: %v", failcount)
		log.Printf("Number of Successes: %v", okcount)
		log.Printf("Success percent: %v%v", (float64(okcount)/float64(failcount+okcount))*100.0, "%")
		log.Printf("Players matched: %v", okcount*16)
		log.Printf("Throughput: %v pps", float64(okcount*16)/time.Since(start).Seconds())

	case "ListMatches":
		stream, err := client.ListMatches(ctx, &pb.ListMatchesRequest{
			Mmfspec:     req.Mmfspec,
			Matchobject: req.Matchobject,
		})
		if err != nil {
			log.Fatalf("Attempting to open stream for ListMatches(_) = _, %v", err)
		}
		for {
			log.Printf("Waiting for matches...")
			match, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("Error reading stream for ListMatches(_) = _, %v", err)
				break
			}
			matchChan <- match
		}
	}

}
