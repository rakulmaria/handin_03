package main

import (
	proto "Handin_03/grpc"
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	clientPort               = flag.Int("cPort", 0, "client port number")
	serverPort               = flag.Int("sPort", 0, "server port number (should match the port used for the server)")
	clientName               = flag.String("name", "John", "name of the client")
	clientLamportClock int64 = 0
)

var JoinClient proto.ChittyChatClient
var wait *sync.WaitGroup

func init() {
	wait = &sync.WaitGroup{}

}

func main() {

	// Parse the flags to get the port for the client
	flag.Parse()

	// Dial the server at the specified port.
	conn, err := grpc.Dial("localhost:"+strconv.Itoa(*serverPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Could not connect to port %d", *serverPort)
	} else {
		log.Printf("Connected to the server at port %d\n", *serverPort)
	}

	JoinClient = proto.NewChittyChatClient(conn)

	// Create a client
	client := &proto.Client{
		Name: *clientName,
	}

	connectToServer(client)
	//fmt.Println("printing clientname: ", client.Name)

	done := make(chan int)
	// Wait for the client (user) to ask for the time
	wait.Add(1)
	go func() {
		defer wait.Done()

		//now we want to scan the input of the user. Meaning the messages they want to send
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input := scanner.Text()
			if input == "exit" {
				JoinClient.LeaveChat(context.Background(), &proto.Connect{
					Client: client,
					Active: false,
				})

				LeaveMessage := &proto.ChatMessage{
					ClientName: client.Name,
					Message:    "client with name " + client.Name + " left the chat",
					Timestamp:  clientLamportClock,
				}
				_, err := JoinClient.Publish(context.Background(), LeaveMessage)
				if err != nil {
					log.Printf(err.Error())
				}
				<-done

			} else {
				//if the input was not exist we sent the message normally
				clientLamportClock++

				message := &proto.ChatMessage{
					ClientName: client.Name,
					Message:    input,
					Timestamp:  clientLamportClock,
				}

				// calling the publish function that takes a message and returns an empty message
				_, err := JoinClient.Publish(context.Background(), message)
				if err != nil {
					log.Printf(err.Error())
				}
			}
		}
	}()

	go func() {
		wait.Wait()
		close(done)
	}()

	//this makes sure that we can only return from the function after the goroutines are done.
	<-done
}

func connectToServer(client *proto.Client) error {

	var streamError error

	stream, err := JoinClient.JoinChat(context.Background(), &proto.Connect{
		Client: client,
		Active: true,
	})

	if err != nil {
		return fmt.Errorf("Connection failed: %v", err)
	} else {
		//log.Printf("client with id %d joined the chat",client.Id)
	}

	wait.Add(1)
	go func(str proto.ChittyChat_JoinChatClient) {
		defer wait.Done()

		for {
			//call to the publishClients method Recv() which returns a chatMessage
			//wait for us to recieve a message from the server
			message, err := str.Recv()

			if message.Timestamp > clientLamportClock {
				clientLamportClock = message.Timestamp
			}

			clientLamportClock++

			//if we don't get a messsage
			if err != nil {
				/// i dont know
				streamError = fmt.Errorf("error reading the message: %v", err)
			}
			//if no error we want to print the message to all clients:
			log.Printf("%v : %s", message.ClientName, message.Message)

		}
	}(stream) //calling the function with stream

	return streamError
}
