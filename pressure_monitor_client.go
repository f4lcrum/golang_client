// Package main implements a client for Greeter service.
package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"os"
	"strconv"
	"sync"
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
	connection        PressureMonitorClient
}

var (
	addr = flag.String("addr", "localhost:5097", "the address to connect to")
	name = flag.String("name", defaultName, "Name to greet")
)

func sendData(mess string, client Client) {
	if mess == "END" {
		client.connection.HangUp(context.Background(), &SetId{Id: client.id.String()})
	}
	n, _ := strconv.ParseFloat(mess, 32)
	num := float32(n)
	client.connection.SetPressure(context.Background(), &SetMessage{NominalPressure: num})
}

//func run(client Client) {
//	var input_message = ""
//	var response_message = ""
//
//	for true {
//		if client.cancelToken == false {
//			if len(client.inputQueue) > 0 {
//				input_message = client.inputQueue[0]
//			}
//		}
//	}
//}

func setPressure(client Client, group sync.WaitGroup) {
	fmt.Println("Usage: '+' for increment, '-' for decrement, 'q' for quit")
	var message = ""
	var input []byte = make([]byte, 1)
	for client.hangUpStart.IsZero() && client.cancelToken == false {

		//reader := bufio.NewReader(os.Stdin)
		//input, err := reader.ReadString('\n')
		os.Stdin.Read(input)
		if string(input) == "+" {
			message = "10"
		} else if string(input) == "-" {
			message = "-10"
		} else if string(input) == "q" {
			message = "END"
			client.hangUpStart = time.Now()
		} else {
			if !client.cancelToken {
				log.Println("+, - or q only!!!")
			}
			continue
		}
		sendData(message, client)
		//client.inputQueue = append(client.inputQueue, message)
	}
	group.Done()
}

//func setPressure2(client Client) {
//	log.Println("cislo: ", 10)
//	client.connection.SetPressure(context.Background(), &SetMessage{NominalPressure: 10})
//}

func listenStream(client Client, group sync.WaitGroup) {
	responseStream, err := client.connection.GetPressureStream(context.Background(), &SetId{Id: client.id.String()})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	for {
		response, err := responseStream.Recv()
		if err != nil {
			if err == io.EOF {
				//log.Debug("client hang up")
				break
			}
			//log.Warning("event receive error occured %v", err)
			break
		}
		fmt.Println("Received from async generator: ", response.NominalPressure, response.CurrentPressure)
	}
	group.Done()
}

func main() {
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	var wg = sync.WaitGroup{}
	wg.Add(1)
	var client = Client{cancelToken: false, recQueue: []string{}, inputQueue: []string{}, isTerminationMess: false, id: uuid.New(), connection: NewPressureMonitorClient(conn)}
	go setPressure(client, wg)
	go listenStream(client, wg)
}
