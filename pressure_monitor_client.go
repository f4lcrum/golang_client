// Package main implements a client for Greeter service.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"io"
	"log"
	"os"
	"time"
)

const (
	defaultName = "world"
)

type Client struct {
	hangUpStart       time.Time
	cancelToken       bool
	recQueue          []string
	inputQueue        []string
	isTerminationMess bool
	id                uuid.UUID
}

var (
	addr = flag.String("addr", "localhost:5097", "the address to connect to")
	name = flag.String("name", defaultName, "Name to greet")
)

func setPressure(client Client) {
	fmt.Println("Usage: '+' for increment, '-' for decrement, 'q' for quit")
	var message = ""
	for client.hangUpStart.IsZero() && client.cancelToken == false {
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if input == "+" {
			message = "10"
		} else if input == "-" {
			message = "-10"
		} else if input == "q" || err == io.EOF {
			message = "END"
			client.hangUpStart = time.Now()
		} else {
			if !client.cancelToken {
				log.Println("+, - or q only!!!")
			}
			continue
		}
		client.inputQueue = append(client.inputQueue, message)
	}

}

func listenStream(conn *grpc.ClientConn, client Client) {
	var c = NewPressureMonitorClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := c.GetPressureStream(ctx, &SetId{Id: client.id.String()})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	setPressure(client)
}

func main() {
	flag.Parse()
	// Set up a connection to the server.
	var client = Client{cancelToken: false, recQueue: []string{}, inputQueue: []string{}, isTerminationMess: false, id: uuid.New()}
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(nil))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	listenStream(conn, client)
}
