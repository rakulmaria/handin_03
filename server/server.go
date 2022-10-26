package main

import (
	proto "Handin_03/grpc"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"

	"google.golang.org/grpc"
)

type Connection struct {
	stream proto.ChittyChat_JoinChatServer
	id     int64
	active bool
	error  chan error
}

// Struct that will be used to represent the Server.
type Server struct {
	proto.UnimplementedChittyChatServer // Necessary
	name                                string
	port                                int
	connection                          []*Connection
}

// Used to get the user-defined port for the server from the command line
var (
	port                     = flag.Int("port", 0, "server port number")
	serverLamportClock int64 = 0
)

func main() {
	// Get the port from the command line when the server is run
	flag.Parse()

	//empty array of connections for the server
	var connections []*Connection

	// Create a server struct
	server := &Server{
		name:       "serverName",
		port:       *port,
		connection: connections,
	}

	// Start the server
	go startServer(server)

	// Keep the server running until it is manually quit
	for {

	}
}

func startServer(server *Server) {

	// Create a new grpc server
	grpcServer := grpc.NewServer()

	// Make the server listen at the given port (convert int port to string)
	listener, err := net.Listen("tcp", "localhost:"+strconv.Itoa(server.port))

	if err != nil {
		log.Fatalf("Could not create the server %v", err)
	}
	log.Printf("Started server at port: %d\n", server.port)

	// Register the grpc server and serve its listener
	proto.RegisterChittyChatServer(grpcServer, server)
	serveError := grpcServer.Serve(listener)
	if serveError != nil {
		log.Fatalf("Could not serve listener")
	}
}

func (s *Server) JoinChat(in *proto.Connect, stream proto.ChittyChat_JoinChatServer) error {

	conn := &Connection{
		stream: stream,
		id:     in.Client.Id,
		active: true,
		error:  make(chan error),
	}

	//putting the new connection into our servers field connections (array of connections)
	s.connection = append(s.connection, conn)

	//we want for every connection to broadcast a message to all saying that a person joined the chat
	serverLamportClock++

	joinedMessage := &proto.ChatMessage{
		ClientId:  in.Client.Id,
		Message:   "\n **** NEW USER JOINED **** \n - " + in.Client.Name + " joined the chat ",
		Timestamp: serverLamportClock,
	}

	s.Publish(conn.stream.Context(), joinedMessage)

	//returning the error if any
	return <-conn.error
}

func (s *Server) Publish(ctx context.Context, in *proto.ChatMessage) (*proto.Empty, error) {
	//counter of go routines. Waits for the go-routines to finish, then decrement the counter
	wait := sync.WaitGroup{}

	//use this to know if our goroutines are finished
	done := make(chan int)

	//this method should for each connection in the servers array of connections, send the message
	for _, conn := range s.connection {
		wait.Add(1)

		go func(message *proto.ChatMessage, conn *Connection) {
			//when the go routine is finished running the counter will decrement
			defer wait.Done()

			if conn.active {
				if message.Timestamp > serverLamportClock {
					serverLamportClock = message.Timestamp
				}

				toBeSentMessage := &proto.ChatMessage{
					ClientId:  conn.id,
					Message:   "\n **** CURRENT LAMPORT TIME **** \n - " + strconv.FormatInt(message.Timestamp, 10) + message.Message,
					Timestamp: serverLamportClock,
				}

				err := conn.stream.Send(toBeSentMessage)

				//if we fail to send a message to the stream. We set the connection to not active
				if err != nil {
					log.Printf("Error with Stream: %v - Error: %v", conn.stream, err)
					conn.active = false
					conn.error <- err
				}
			}
		}(in, conn)
	}
	go func() {
		wait.Wait()
		close(done)
	}()

	fmt.Println("Printing number of connections made: ", len(s.connection))
	//this makes sure that we can only return from the function after the goroutines are done.
	<-done
	return &proto.Empty{}, nil
}
