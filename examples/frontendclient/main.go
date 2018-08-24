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
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

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
	client := frontend.NewAPIClient(conn)
	log.Println("API client connected!")

	log.Printf("Establishing HTTPv2 stream...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Empty group to fill and then run through the CreateRequest gRPC endpoint
	g := &frontend.Group{
		Id:         "",
		Properties: "",
	}

	// Generate players for the group and put them in
	for i := 0; i < numPlayers; i++ {
		playerID, playerData, debug := player.Generate()
		groupPlayer(g, playerID, playerData)
		_ = debug // TODO.  For now you could copy this into playerdata before creating player if you want it available in redis
		pretty.PrettyPrint(playerID)
		pretty.PrettyPrint(playerData)
	}
	g.Id = g.Id[:len(g.Id)-1] // Remove trailing whitespace

	log.Printf("Finished grouping players")

	// Test CreateRequest
	log.Println("Testing CreateRequest")
	results, err := client.CreateRequest(ctx, g)
	if err != nil {
		panic(err)
	}

	pretty.PrettyPrint(g.Id)
	pretty.PrettyPrint(g.Properties)
	pretty.PrettyPrint(results.Success)

	// wait for value to  be inserted that will be returned by Get ssignment
	test := "bitters"
	fmt.Println("Pausing: go put a value to return in Redis using HSET", test, "connstring <YOUR_TEST_STRING>")
	fmt.Println("Hit Enter to test GetAssignment...")
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
	connstring, err := client.GetAssignment(ctx, &frontend.PlayerId{Id: test})
	pretty.PrettyPrint(connstring.ConnectionString)

	// Test DeleteRequest
	fmt.Println("Deleting Request")
	results, err = client.DeleteRequest(ctx, g)
	pretty.PrettyPrint(results.Success)

	// Remove assignments key
	fmt.Println("deleting the key", test)
	results, err = client.DeleteAssignment(ctx, &frontend.PlayerId{Id: test})
	pretty.PrettyPrint(results.Success)

	return
}

func groupPlayer(g *frontend.Group, playerID string, playerData map[string]int) error {
	//g.Properties = playerData
	pdJSON, _ := json.Marshal(playerData)
	buffer := new(bytes.Buffer) // convert byte array to buffer to send to json.Compact()
	if err := json.Compact(buffer, pdJSON); err != nil {
		log.Println(err)
	}
	g.Id = g.Id + playerID + " "

	// TODO: actually aggregate group stats
	g.Properties = buffer.String()
	return nil
}
