/*
This is a testing program that stubs multiple frontend api clients, all
requesting matchmaking as a group. This assumes it is run within a k8s cluster,
and that the frontend api is up and can be accessed through a k8s service
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
	"flag"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/GoogleCloudPlatform/open-match/test/cmd/frontendclient/player"
	"github.com/gobs/pretty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

var (
	numPlayers = flag.Int("numplayers", 1, "number of players to generate and group together")
	connect    = flag.Bool("connect", false, "Set to true to try to connect to the assigned server over UDP")
	noDelete   = flag.Bool("no-delete", false, "Set to true to prevent deletion of request and player info when done")
	server     = flag.String("frontend", "om-frontendapi:50504", "Hostname or IP of the Open Match frontend")
	cycle      = flag.Bool("cycle", true, "Continuously add more players.")
)

func main() {
	// Get command line flags
	flag.Parse()
	log.Print("Parsing flags:")
	log.Printf(" [flags] Connecting to frontend API at %v", *server)
	log.Printf(" [flags] Generating %d players", *numPlayers)
	log.Printf(" [flags] Attempt connect to DGS? %v", *connect)
	log.Printf(" [flags] Leave player and request in state storage? %v", *noDelete)

	// Connect gRPC client
	conn, err := grpc.Dial(*server, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err.Error())
	}
	client := pb.NewFrontendClient(conn)
	log.Println("API client connected!")
	if *cycle {
		for {
			addPlayer(client)
		}
	} else {
		addPlayer(client)
	}
}

func addPlayer(client pb.FrontendClient) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Open Match doesn't have a separate concept of 'groups': a group is
	// represented by a moke 'aggregate' player with an ID that is a
	// concatenation of all the players in the group, and properties that you
	// choose to represent where you want the group to show up in indexed
	// search results.  For example, a common strategy for choosing the
	// attribute values for a group of players might be to average the latency
	// of all players, and average the MMR/SR and then add a grouping 'bonus'
	// under the assumption that they will play better together than a random
	// selection of players.
	//
	// To request a match for a group, you first create this 'aggregate' player and
	// then request matchmaking for the group using CreateRequest().  Then,
	// each player should individually call GetAssignment() with their single
	// ID to receive updates.

	// Empty group to fill and then send the CreateRequest gRPC endpoint
	g := &pb.Player{
		Id:         "",
		Properties: "",
	}

	// Generate players for the group
	player.New() // Player generator init
	players := make([]*pb.Player, *numPlayers)
	resultsChan := make(chan pb.Player)
	for i := 0; i < *numPlayers; i++ {
		// Var init
		players[i] = &pb.Player{}

		// Generate a new player
		newPlayerID, playerData := player.Generate()
		players[i].Id = newPlayerID
		players[i].Properties = playerDataToJSONString(playerData)

		// Start watching for results for this player (assignment/status/error)
		responseStream, err := client.GetUpdates(ctx, &pb.GetUpdatesRequest{
			Player: players[i],
		})
		if err != nil {
			log.Printf("Error on GetUpdates(ID=%v), %s\n", players[i].Id, err)
		}
		go waitForResults(resultsChan, responseStream)
		log.Printf("Generated player \"%v\": %v", players[i].Id, players[i].Properties)

		if *numPlayers > 1 {
			// Add the new player to the mock 'aggregate' player to represent the group
			groupPlayer(g, players[i].Id, playerData)
		}
	}

	if *numPlayers > 1 {
		g.Id = g.Id[:len(g.Id)-1] // Remove trailing whitespace
		log.Printf("Finished grouping players into mock 'aggregate' player: \"%v\"", g.Id)
	} else {
		g = players[0]
	}

	// Test CreateRequest
	log.Println("Testing CreatePlayer")
	results, err := client.CreatePlayer(ctx, &pb.CreatePlayerRequest{
		Player: g,
	})
	if err != nil {
		log.Fatal(err)
	}
	if !*noDelete {
		if *numPlayers > 1 {
			defer cleanup(client, g, "group")
		}
		for _, p := range players {
			defer cleanup(client, p, "player")
		}
	}
	log.Printf("CreateRequest() returned %v", results)

	if !*cycle {
		// Get updates for each player and display them the 'q' key is pressed.
		quitChan := make(chan bool)
		go waitForQuit(quitChan)
		quit := false
		for !quit {
			log.Println("Waiting for results from Open Match...")
			log.Println("**Press Enter to disconnect from Open Match frontend**")

			select {
			case a := <-resultsChan:
				pretty.PrettyPrint(a)
			case <-quitChan:
				log.Println("Disconnect from Frontend requested")
				cancel() // Quit listening for results.
				quit = true
			}
		}
	}

	// Assignment received, run a single dumb UDP client that just prints to the
	// screen the contents of the packets it recieves
	player := players[0]
	if *connect && player.Assignment != "" {
		err = udpClient(context.Background(), player.Assignment)
		if err != nil {
			log.Printf("Error on udpClient, %s\n", errors.WithStack(err))
		}
	}
}

func waitForQuit(quitChan chan<- bool) {
	// Don't do anything with the input; any input means to quit.
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
	quitChan <- true
}

// waitForResults loops until the context is cancelled, looking for assignment/status/error updates for a player.
func waitForResults(resultsChan chan pb.Player, stream pb.Frontend_GetUpdatesClient) {
	for {
		a, err := stream.Recv()
		if err == io.EOF {
			log.Println("io.EOF recieved")
			break
		}
		if err != nil {
			if strings.HasSuffix(err.Error(), "context canceled") {
				break
			}
			log.Println("Error encountered")
			log.Printf("Error reading stream for GetAssignments(_) = _, %v", err)
			break
		}
		log.Println("Result recieved from Open Match!")
		resultsChan <- *a.Player
	}
}

func udpClient(ctx context.Context, address string) error {
	// Resolve address
	raddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return errors.WithStack(err)
	}

	// Connect
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return errors.WithStack(err)
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
			err = errors.WithStack(err)
			doneChan <- err
			return
		}

		// Loop forever, reading
		buffer := make([]byte, 1024)
		for {
			n, _, err := conn.ReadFrom(buffer)
			if err != nil {
				err = errors.WithStack(err)
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
		return err
	case err = <-doneChan:
		return err
	}
}

func cleanup(client pb.FrontendClient, g *pb.Player, kind string) {
	// Test DeleteRequest
	results, err := client.DeletePlayer(context.Background(), &pb.DeletePlayerRequest{
		Player: g,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Deleting Request for %v %v: %v", kind, g.Id, results)
}

func groupPlayer(g *pb.Player, playerID string, playerData map[string]interface{}) error {
	// Concatinate this player to the existing group
	g.Id = g.Id + playerID + " "

	// TODO: This just overwrites the group's 'aggregate' player attributes
	// with the attributes of the last player we generated. In a real game
	// client, you would need to choose the way you want to represent multiple
	// players in your searchable indexes.  See more details in the main()
	// function comments.
	g.Properties = playerDataToJSONString(playerData)

	return nil
}

func playerDataToJSONString(playerData map[string]interface{}) string {
	pdJSON, _ := json.Marshal(playerData)
	buffer := new(bytes.Buffer) // convert byte array to buffer to send to json.Compact()
	if err := json.Compact(buffer, pdJSON); err != nil {
		log.Println(err)
	}

	return buffer.String()
}
