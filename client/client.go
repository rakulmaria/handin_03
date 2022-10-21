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
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	id         int64
	portNumber int
}

var (
	clientPort = flag.Int("cPort", 0, "client port number")
	serverPort = flag.Int("sPort", 0, "server port number (should match the port used for the server)")
)

var publishClient proto.ChittyChatClient
var wait *sync.WaitGroup

func init(){
	wait = &sync.WaitGroup{}
}


func main() {

	//timestamp := time.Now()
	done := make(chan int)
	// Parse the flags to get the port for the client
	flag.Parse()

	name := flag.String("N", "Anon", "The name of the user")
	id := int64(1)


	// Dial the server at the specified port.
	conn, err := grpc.Dial("localhost:"+strconv.Itoa(*serverPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Could not connect to port %d", *serverPort)
	} else {
		log.Printf("Connected to the server at port %d\n", *serverPort)
	}

	publishClient = proto.NewChittyChatClient(conn)

	// Create a client
	client := &proto.Client{
		Id: id,
		Name: *name,
	}

	connectToServer(client)


	// Wait for the client (user) to ask for the time
	wait.Add(1)
	go func(){
		defer wait.Done()

		//now we want to scan the input of the user. Meaning the messages they want to send
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input := scanner.Text()

			message := &proto.ChatMessage{
				ClientId: client.Id,
				Message: input,
				Timestamp: time.Now().String(),
			}

				// calling the broadcast function that takes a message and returns an empty message
			_, err := publishClient.Broadcast(context.Background(),message)
			if err != nil {
				log.Printf(err.Error())
			} 
			log.Printf("client with id: %v sent the message: %s",message.ClientId, message.Message)
		}
	}()

	go func(){
		wait.Wait()
		close(done)
	}()

	//this makes sure that we can only return from the function after the goroutines are done. 
		<-done
}

func connectToServer(client *proto.Client) error {

	var streamError error

	stream, err := publishClient.Publish(context.Background(), &proto.Connect{
		Client:   client,
		Active: true,
	})

	if err != nil{
		return fmt.Errorf("Connection failed: %v",err)
	}

	wait.Add(1)
		go func(str proto.ChittyChat_PublishClient){
			defer wait.Done()

			for {
				//call to the publish clients method
				//wait for us to recieve a message from the server
				message,err := str.Recv()

				//if we don't get a messsage
				if err != nil{
					/// i dont know
					streamError = fmt.Errorf("error reading the message: %v", err)
				}
				//if no error we want to print the message to all clients: 
				log.Printf("%v : %s", message.ClientId, message.Message)

			}
		}(stream) //calling the function with stream


	return streamError
}




	