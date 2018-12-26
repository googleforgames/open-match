/*
Stubbed frontend api client. This should be run within a k8s cluster, and
assumes that the frontend api is up and can be accessed through a k8s service
called 'om-frontendapi'.

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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/open-match/examples/frontendclient/player"
	frontend "github.com/GoogleCloudPlatform/open-match/examples/frontendclient/proto"
	"github.com/gobs/pretty"
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

	// determine number of players to generate per group
	numPlayers := 4 // default if nothing provided
	var err error
	// Set to true if you don't have an MMF set up to find this player and want
	// to manually enter their assignment in redis
	var manualAssignment bool = true

	// Try to read number of players from command line.
	if len(os.Args) > 1 {
		numPlayers, err = strconv.Atoi(os.Args[1])
		if err != nil {
			panic(err)
		}

	}
	player.New()
	log.Printf("Generating %d players", numPlayers)

	// Connect gRPC client
	ip, err := net.LookupHost("om-frontendapi")
	if err != nil {
		panic(err)
	}
	_ = ip
	conn, err := grpc.Dial(ip[0]+":50504", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err.Error())
	}
	client := frontend.NewFrontendClient(conn)
	log.Println("API client connected!")

	log.Printf("Establishing HTTPv2 stream...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Empty group to fill and then run through the CreateRequest gRPC endpoint
	g := &frontend.Player{
		Id:         "",
		Properties: "",
	}

	// Generate players for the group and put them in
	for i := 0; i < numPlayers; i++ {
		playerID, playerData, debug := player.Generate()
		groupPlayer(g, playerID, playerData)
		_ = debug // TODO.  For now you could copy this into playerdata before creating player if you want it available in redis
		log.Printf("%v: %v", g.Id, g.Properties)
	}
	g.Id = g.Id[:len(g.Id)-1] // Remove trailing whitespace

	log.Printf("Finished grouping players")

	// Test CreateRequest
	log.Println("Testing CreateRequest")
	results, err := client.CreateRequest(ctx, g)
	//defer cleanup(ctx, client, g)
	if err != nil {
		panic(err)
	}

	_ = results

	// wait for value to  be inserted that will be returned by Get ssignment
	test := strings.Split(g.Id, " ")[0] // First player ID will be the one we look in.
	if manualAssignment {
		log.Println("Pausing: go put a value to return in Redis using:")
		log.Println("HSET", test, "assignment <YOUR_TEST_STRING>")
		log.Println("Hit Enter to test GetAssignment...")
		reader := bufio.NewReader(os.Stdin)
		_, _ = reader.ReadString('\n')
	}

	// Endless rety to get an assignment until one is received or this process is killed.
	a := &frontend.Player{}
	for len(a.Assignment) == 0 {
		stream, err := client.GetAssignment(ctx, &frontend.Player{Id: test})
		for {
			log.Println("No assignment recieved from Open Match yet")
			a, err = stream.Recv()
			if err == io.EOF {
				log.Println("io.EOF recieved")
				break
			}
			if err != nil {
				log.Println("Error encountered")
				log.Println("Error reading stream for GetAssignments(_) = _, %v", err)
				break
			}
			if len(a.Assignment) > 0 {
				log.Println("Result recieved from Open Match!")
				pretty.PrettyPrint(a)
				log.Println("To stop getting updates, go put this in Redis:")
				log.Println("HSET", test, "status break")
				if a.Status == "break" {
					break
				}
			}
		}
	}

	// Assignment received, run UDP client and try to receive timestamps.
	err = udpClient(context.Background(), a.Assignment)
	if err != nil {
		panic(err)
	}

	return
}

func udpClient(ctx context.Context, address string) (err error) {
	// Resolve address
	raddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return
	}

	// Connect
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return
	}
	defer conn.Close()

	// Subscribe and loop, printing received timestamps
	doneChan := make(chan error, 1)
	go func() {
		// Send one packet to servers so it will start sending us timestamps
		log.Println("Connecting...")
		b := bytes.NewBufferString("subscribe")
		_, err := io.Copy(conn, b)
		if err != nil {
			doneChan <- err
			return
		}

		// Loop forever, reading
		buffer := make([]byte, 1024)
		for {
			n, _, err := conn.ReadFrom(buffer)
			if err != nil {
				doneChan <- err
				return
			}

			log.Printf("received %v\n", string(buffer[:n]))

		}
	}()

	// Exit if listening loop has an error or context is cancelled
	select {
	case <-ctx.Done():
		log.Println("context cancelled")
		err = ctx.Err()
	case err = <-doneChan:
	}

	return
}

func cleanup(ctx context.Context, client frontend.FrontendClient, g *frontend.Player) {
	// Test DeleteRequest
	log.Println("Deleting Request")
	results, err := client.DeleteRequest(ctx, g)
	if err != nil {
		log.Fatal(err)
	}
	pretty.PrettyPrint(results.Success)
}

func groupPlayer(g *frontend.Player, playerID string, playerData map[string]int) error {
	//g.Properties = playerData
	pdJSON, _ := json.Marshal(playerData)
	buffer := new(bytes.Buffer) // convert byte array to buffer to send to json.Compact()
	if err := json.Compact(buffer, pdJSON); err != nil {
		log.Println(err)
	}
	g.Id = g.Id + playerID + " "

	// TODO: actually aggregate group stats
	g.Properties = buffer.String()
	/*
		err := errors.New("")
		g.Properties, err = strconv.Unquote(string(pdJSON))
		log.Println(buffer.String())
		log.Println("==========")
		g.Properties, err = strconv.Unquote(buffer.String())
		if err != nil {
			log.Println("Issue unquoting")
			log.Println(buffer.String())
			log.Println(err)
		}
	*/

	return nil
}
