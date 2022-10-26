package main

import (
	proto "Handin_03/grpc"
	"context"
	"flag"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
)

type Connection struct {
	stream proto.ChittyChat_JoinChatServer
	name   string
	active bool
	error  chan error
}

// Struct that will be used to represent the Server.
type Server struct {
	proto.UnimplementedChittyChatServer // Necessary
	name                                string
	port                                int
	//connection                          []*Connection
	users map[string]*Connection
}

// Used to get the user-defined port for the server from the command line
var port = flag.Int("port", 0, "server port number")
var users = make(map[string]*Connection)

func main() {

	//setting the log file
	f, err := os.OpenFile("golang-demo.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
    	log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()	
	log.SetOutput(f)


	// Get the port from the command line when the server is run
	flag.Parse()

	// Create a server struct
	server := &Server{
		name: "serverName",
		port: *port,
		users: users,
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
		name:   in.Client.Name,
		active: true,
		error:  make(chan error),
	}

	//putting the new connection into our servers field connections (map of clientname and connections)
	s.users[in.Client.Name] = conn

	//we want for every connection to broadcast a message to all saying that a person joined the chat
	joinedMessage := &proto.ChatMessage{
		ClientName: in.Client.Name,
		Message:    "client with name " + in.Client.Name + " joined the chat",
		Timestamp:  time.Now().String(),
	}

	for _, conn := range s.users {
		s.Publish(conn.stream.Context(), joinedMessage) //WHY IS THIS PRINTING SOOO MANY TIMES??
	}

	log.Printf("Participant " + in.Client.Name + " joined the Chitty-chat at lamport time xxxxx")

	//returning the error if any
	return <-conn.error
}

func (s *Server) Publish(ctx context.Context, in *proto.ChatMessage) (*proto.Empty, error) {
	//counter of go routines. Waits for the go-routines to finish, then decrement the counter
	wait := sync.WaitGroup{}

	//use this to know if our goroutines are finished
	done := make(chan int)

	//this method should for each connection in the servers array of connections, send the message
	for _, conn := range s.users {
		wait.Add(1)

		go func(message *proto.ChatMessage, conn *Connection) {
			//when the go routine is finished running the counter will decrement
			defer wait.Done()

			if conn.active {
				err := conn.stream.Send(message)

				//if we fail to send a message to the stream. We set the connection to not active
				if err != nil {
					//log.Printf("Error with Stream: %v - Error: %v", conn.stream, err)
					conn.active = false
					conn.error <- err
				}
			}  else {
				log.Printf("participant: " + message.ClientName + "left Chitty-Chat at lamport time")
			}
		}(in, conn)
	}
	go func() {
		wait.Wait()
		close(done)
	}()
	//this makes sure that we can only return from the function after the goroutines are done.
	<-done
	return &proto.Empty{}, nil
}

func (s *Server) LeaveChat(in *proto.Connect, stream proto.ChittyChat_LeaveChatServer) error {
	for name := range s.users {
		if name == in.Client.Name {
			delete(s.users, name)
			log.Printf("participant: " + in.Client.Name + "left Chitty-Chat at lamport time")
		}
	}
	return nil
}
