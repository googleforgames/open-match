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
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/GoogleCloudPlatform/open-match/test/cmd/frontendclient/player"
	"github.com/gobs/pretty"
	"google.golang.org/grpc"
)

var (
	numPlayers = flag.Int("numplayers", 1, "number of players to generate and group together")
	connect    = flag.Bool("connect", false, "Set to true to try to connect to the assigned server over UDP")
	noDelete   = flag.Bool("no-delete", false, "Set to true to prevent deletion of request and player info when done")
	server     = flag.String("frontend", "om-frontendapi", "Hostname or IP of the Open Match frontend")
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
	ip, err := net.LookupHost(*server)
	if err != nil {
		panic(err)
	}
	_ = ip
	conn, err := grpc.Dial(ip[0]+":50504", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err.Error())
	}
	client := pb.NewFrontendClient(conn)
	log.Println("API client connected!")
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
		playerData := make(map[string]interface{})

		// Generate a new player
		players[i].Id, playerData = player.Generate()
		players[i].Properties = playerDataToJSONString(playerData)

		// Start watching for results for this player (assignment/status/error)
		responseStream, _ := client.GetUpdates(ctx, &pb.GetUpdatesRequest{
			Player: players[i],
		})
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
	if !*noDelete {
		if *numPlayers > 1 {
			defer cleanup(client, g, "group")
		}
		for _, p := range players {
			defer cleanup(client, p, "player")
		}
	}
	if err != nil {
		panic(err)
	}
	log.Printf("CreateRequest() returned %v", results)

	// Get updates for each player and display them until the 'enter' key is pressed.
	quitChan := make(chan struct{})
	go func(q chan<- struct{}) {
		r := bufio.NewReader(os.Stdin)
		r.ReadString('\n')
		close(q)
	}(quitChan)
L:
	for {
		log.Println("Waiting for results from Open Match...")
		log.Println("**Press Enter to disconnect from Open Match frontend**")

		select {
		case <-quitChan:
			log.Println("Disconnect from Frontend requested")
			cancel() // Quit listening for results.
			break L

		case a := <-resultsChan:
			pretty.PrettyPrint(a)
			if a.Pool != "" {
				players[0].Pool = a.Pool
			}
			if a.Status != "" {
				players[0].Status = a.Status
			}
			if a.Error != "" {
				players[0].Error = a.Error
			}
			if a.Properties != "" {
				players[0].Properties = a.Properties
			}
			if a.Assignment != "" {
				players[0].Assignment = a.Assignment
			}
		}
	}

	// Assignment received, run a single dumb UDP client that just prints to the
	// screen the contents of the packets it recieves
	player := players[0]
	if *connect && player.Assignment != "" {
		udpCtx, stopUDP := context.WithCancel(context.Background())

		cmdChan := make(chan string)
		go func() {
			waitForInput(cmdChan)
			stopUDP()
		}()

		err = udpClient(udpCtx, player.Assignment, player.Id, cmdChan)
		if err != nil {
			panic(err)
		}
	}

	return
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
			log.Printf("Error reading stream for GetAssignments(_) = _, %v\n", err)
			break
		}
		log.Println("Result recieved from Open Match!")
		resultsChan <- *a.Player
	}
}

func waitForInput(cmdChan chan string) {
	helpText := `(Type commands; start it with "<-" to send to server; type "QUIT" to terminate communication and exit; messages received from server start with "->";)`
	fmt.Println(helpText)

	for {
		r := bufio.NewReader(os.Stdin)
		line, err := r.ReadString('\n')
		if err != nil {
			panic(err)
		}

		s := strings.TrimSpace(line)
		switch {
		case strings.ToUpper(s) == "QUIT":
			return

		case strings.HasPrefix(s, "<-"):
			cmd := strings.TrimSpace(strings.TrimPrefix(s, "<-"))
			cmdChan <- cmd

		default:
			fmt.Println(helpText)
		}
	}
}

func udpClient(ctx context.Context, address, playerID string, cmdChan <-chan string) (err error) {
	log.Printf("Connecting to udp-server at %s...\n", address)
	raddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return
	}
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return
	}
	defer conn.Close()

	errChan := make(chan error, 2)
	doneChan := make(chan struct{})

	go func() {
		// Loop forever, reading
		buffer := make([]byte, 1024)
		for {
			select {
			case <-doneChan:
				return
			default:
				n, _, err := conn.ReadFrom(buffer)
				if err != nil {
					errChan <- err
					return
				}
				s := string(buffer[:n])
				fmt.Println("->", s)
			}
		}
	}()

	go func() {
		// Send the user input to server
		for {
			select {
			case <-doneChan:
				return
			case cmd := <-cmdChan:
				msg := cmd
				if !strings.HasPrefix(cmd, playerID) {
					msg = playerID + " " + cmd
				}
				log.Printf("sending \"%v\"", msg)
				_, err := conn.Write([]byte(msg))
				if err != nil {
					errChan <- err
					return
				}
			}
		}
	}()

	// Wait for error or context cancelation
	select {
	case <-ctx.Done():
		log.Println("UDP client context cancelled")
		if ctx.Err() != context.Canceled {
			err = ctx.Err()
		}
	case err = <-errChan:
	}

	// Stop reading and wiriting goroutines
	close(doneChan)
	return
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
