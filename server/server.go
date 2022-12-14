package main

import (
	proto "Handin_03/grpc"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"

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
	users                               map[string]*Connection
}

// Used to get the user-defined port for the server from the command line
var (
	port                     = flag.Int("port", 0, "server port number")
	serverLamportClock int64 = 0
	users                    = make(map[string]*Connection)
)

func main() {

	//setting the log file
	f, err := os.OpenFile("golang-demo.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// Get the port from the command line when the server is run
	flag.Parse()

	// Create a server struct
	server := &Server{
		name:  "serverName",
		port:  *port,
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
	log.Printf("Started server at port: %d\n", server.port) //******************************************LOGGING Started server at...

	// Register the grpc server and serve its listener
	proto.RegisterChittyChatServer(grpcServer, server)
	serveError := grpcServer.Serve(listener)
	if serveError != nil {
		log.Fatalf("Could not serve listener") //******************************************LOGGING Could not server listener...
	}
}

func (s *Server) JoinChat(in *proto.Connect, stream proto.ChittyChat_JoinChatServer) error {

	conn := &Connection{
		stream: stream,
		name:   in.Client.Name,
		active: true,
		error:  make(chan error),
	}

	//putting the new connection into our servers field connections (array of connections)

	s.users[in.Client.Name] = conn

	//we want for every connection to broadcast a message to all saying that a person joined the chat

	joinedMessage := &proto.ChatMessage{
		ClientName: in.Client.Name,
		Message:    "\n ** NEW USER JOINED ** \n - " + in.Client.Name + " joined the chat ",
		Timestamp:  in.Client.ClientClock,
	}

	if serverLamportClock < joinedMessage.Timestamp {
		serverLamportClock = joinedMessage.Timestamp
	}

	s.Publish(conn.stream.Context(), joinedMessage)

	log.Printf("Participant " + in.Client.Name + " joined the Chitty-chat at lamport time " + "%v",serverLamportClock);
	 //******************************************LOGGING Participant x joined the chat at lamport time x...

	//returning the error if any
	return <-conn.error
}

func (s *Server) Publish(ctx context.Context, in *proto.ChatMessage) (*proto.Empty, error) {
	//counter of go routines. Waits for the go-routines to finish, then decrement the counter
	wait := sync.WaitGroup{}

	//use this to know if our goroutines are finished
	done := make(chan int)
	if serverLamportClock < in.Timestamp {
		serverLamportClock = in.Timestamp
	}

	serverLamportClock++

	//this method should for each connection in the servers array of connections, send the message
	for _, conn := range s.users {
		wait.Add(1)
		fmt.Printf("Serverclock in foreach: %v", serverLamportClock)
		go func(message *proto.ChatMessage, conn *Connection) {
			serverLamportClock++
			//when the go routine is finished running the counter will decrement

			defer wait.Done()

			if conn.active {
				toBeSentMessage := &proto.ChatMessage{
					ClientName: in.ClientName, //this needs to be the clientname of the sent message not each client in the range
					Message:    in.ClientName + ": " + message.Message,
					Timestamp:  serverLamportClock,
				}

				toBeSentMessage.Message += "\n"

				err := conn.stream.Send(toBeSentMessage)
				

				//if we fail to send a message to the stream. We set the connection to not active
				if err != nil {
					log.Printf("participant: " + message.ClientName + "left Chitty-Chat at lamport time " + "%v",serverLamportClock) //******************************************LOGGING Participant x left when no connection active....
					conn.active = false
					conn.error <- err
				}
			}
		}(in, conn)
	}
	log.Printf(in.ClientName + " sent message: " + in.Message + " at lamport time: " + "%v",serverLamportClock)

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
			log.Printf("participant " + in.Client.Name + " left the Chitty-Chat at lamport time " + "%v",serverLamportClock) //******************************************LOGGING Participant left through the leavechat function.....

		}
	}
	return nil
}
