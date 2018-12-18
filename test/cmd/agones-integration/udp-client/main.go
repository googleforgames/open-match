package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/gobs/pretty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/GoogleCloudPlatform/open-match/examples/frontendclient/player"
	frontend "github.com/GoogleCloudPlatform/open-match/examples/frontendclient/proto"
)

const maxBufferSize = 1024

func main() {
	// -- start of copied from examples/frontendclient --

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
	g := &frontend.Group{}

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

	// -- end of copied from examples/frontendclient --

	var connstring *frontend.ConnectionInfo

	for i := 0; i < 4; i++ {
		connstring, err = client.GetAssignment(ctx, &frontend.PlayerId{Id: test})
		if err != nil {
			// did not see matchmaking results in redis before timeout
			if st, ok := status.FromError(err); ok {
				if st.Message() == "did not see matchmaking results in redis before timeout" {
					err = nil
					log.Printf("No assignments yet (attempt #%d). Waiting for 15s before retry", i+1)
					<-time.After(15 * time.Second)
				}
			}
		}
		if err != nil {
			panic(err)
		}
	}

	log.Printf("Got connection string: %s", connstring.ConnectionString)
	err = udpClient(context.Background(), connstring.ConnectionString)
	if err != nil {
		panic(err)
	}
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
		fmt.Println("Connecting...")
		b := bytes.NewBufferString("subscribe")
		_, err := io.Copy(conn, b)
		if err != nil {
			doneChan <- err
			return
		}

		// Loop forever, reading
		buffer := make([]byte, maxBufferSize)
		for {
			n, _, err := conn.ReadFrom(buffer)
			if err != nil {
				doneChan <- err
				return
			}

			fmt.Printf("received %v\n", string(buffer[:n]))

		}
	}()

	// Exit if listening loop has an error or context is cancelled
	select {
	case <-ctx.Done():
		fmt.Println("context cancelled")
		err = ctx.Err()
	case err = <-doneChan:
	}

	return
}

func bytesToString(data []byte) string {
	return string(data[:])
}

func ppJSON(s string) {
	buf := new(bytes.Buffer)
	json.Indent(buf, []byte(s), "", "  ")
	log.Println(buf)
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
