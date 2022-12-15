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
	hangUpStart time.Time
	cancelToken bool
	id          uuid.UUID
	connection  PressureMonitorClient
}

var (
	addr = flag.String("addr", "localhost:5097", "the address to connect to")
	name = flag.String("name", defaultName, "Name to greet")
)

var (
	WarningLogger *log.Logger
	InfoLogger    *log.Logger
	ErrorLogger   *log.Logger
)

func sendData(mess string, client *Client) {
	if mess == "END" {
		client.connection.HangUp(context.Background(), &SetId{Id: client.id.String()})
	}
	n, _ := strconv.ParseFloat(mess, 32)
	num := float32(n)
	client.connection.SetPressure(context.Background(), &SetMessage{NominalPressure: num})
}

func setPressure(client *Client, group *sync.WaitGroup, q *chan struct{}) {
	defer group.Done()
	fmt.Println("Usage: '+' for increment, '-' for decrement, 'q' for quit")
	var message = ""
	var input []byte = make([]byte, 1)
	for client.hangUpStart.IsZero() && client.cancelToken == false {

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
	}
	InfoLogger.Println("setPressure ends")
}

func initLog() {
	file, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		ErrorLogger.Println("error occured: %v", err)
		log.Fatal(err)
	}

	InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningLogger = log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func listenStream(client *Client, group *sync.WaitGroup, q *chan struct{}) {
	defer group.Done()
	responseStream, err := client.connection.GetPressureStream(context.Background(), &SetId{Id: client.id.String()})
	if err != nil {
		ErrorLogger.Println("could not greet: %v", err)
	}
	for {
		response, err := responseStream.Recv()
		if err != nil {
			if err == io.EOF {
				WarningLogger.Println("client hang up")
				log.Println("client hang up")
			} else {

			}
			WarningLogger.Println("event receive errror occured ", err)
			break
		}

		if response.LastMessage == true {
			break
		}
		InfoLogger.Println("Received from async generator: ", response.NominalPressure, response.CurrentPressure)
		fmt.Println("Received from async generator: ", response.NominalPressure, response.CurrentPressure)
	}
	client.cancelToken = true
	InfoLogger.Println("listenStream ends")
}

func main() {
	initLog()
	InfoLogger.Println("---------------------------START OF PROGRAM-----------------------------")
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		ErrorLogger.Println("did not connect: %v", err)
		log.Fatalf("did not connect: ", err)
	}
	quit := make(chan struct{})
	var wg sync.WaitGroup
	var client = Client{cancelToken: false, id: uuid.New(), connection: NewPressureMonitorClient(conn)}
	wg.Add(1)
	go setPressure(&client, &wg, &quit)
	wg.Add(1)
	go listenStream(&client, &wg, &quit)

	wg.Wait()
	InfoLogger.Println("-----------------------------END OF PROGRAM------------------------------")
}
