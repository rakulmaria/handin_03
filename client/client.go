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
	clientPort = flag.Int("cPort", 0, "client port number")
	serverPort = flag.Int("sPort", 0, "server port number (should match the port used for the server)")
	clientName = flag.String("name", "John", "name of the client")
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
		//log.Printf("Connected to the server at port %d\n", *serverPort) --------Out commented to clear terminal a bit this is not logged anyway
	}

	JoinClient = proto.NewChittyChatClient(conn)

	// Create a client
	client := &proto.Client{
		Name:        *clientName,
		ClientClock: 0,
	}

	connectToServer(client)

	done := make(chan int)

	// Wait for the client (user) to ask for the time
	wait.Add(1)
	go func() {
		defer wait.Done()

		//now we want to scan the input of the user. Meaning the messages they want to send
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			client.ClientClock++

			input := scanner.Text()
			if input == "exit" {
				JoinClient.LeaveChat(context.Background(), &proto.Connect{
					Client: client,
					Active: false,
				})

				LeaveMessage := &proto.ChatMessage{
					ClientName: client.Name,
					Message:    "** USER LEFT THE CHAT ** \n",
					Timestamp:  client.ClientClock,
				}
				_, err := JoinClient.Publish(context.Background(), LeaveMessage)
				if err != nil {
					log.Printf(err.Error())
				}
				<-done

			} else {
				//if the input was not exist we sent the message normally

				message := &proto.ChatMessage{
					ClientName: client.Name,
					Message:    input,
					Timestamp:  client.ClientClock, // ændre det til client.clock
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

	client.ClientClock++

	stream, err := JoinClient.JoinChat(context.Background(), &proto.Connect{
		Client: client,
		Active: true,
	})

	if err != nil {
		return fmt.Errorf("Connection failed: %v", err)
	}
	wait.Add(1)
	go func(str proto.ChittyChat_JoinChatClient) {
		defer wait.Done()

		for {
			//call to the publishClients method Recv() which returns a chatMessage
			//wait for us to recieve a message from the server
			message, err := str.Recv()

			if message.Timestamp > client.ClientClock {
				client.ClientClock = message.Timestamp
			}

			client.ClientClock++

			//if we don't get a messsage
			if err != nil {
				/// i dont know
				streamError = fmt.Errorf("error reading the message: %v", err)
			}
			//if no error we want to print the message to all clients:
			fmt.Printf("NAME: " + client.Name + " LAMPORT TIME: " + strconv.FormatInt(client.ClientClock, 10) + "\n MESSAGE: " + message.Message)

		}
	}(stream) //calling the function with stream

	return streamError
}
