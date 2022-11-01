# Handin_03

## How to run the Chitty Chat service


### **Setup server**
To run the implemented Chitty-Chat service, open a terminal and start the service by running the following command. (The port no. can be another, just make sure the clients join the same serverPort and run on another clientPort)

    go run server/server.go -port=8081

### **Setup three clients**
Open up three more terminals (for simulating three clients) and run the following commands in each respective terminal. You have to provide the clients a client port, server port and **name**. If you don't provide a name, the name will by default be John.

You can copy and run the following commands.

2nd terminal:

    go run client/client.go -cPort 8082 -sPort 8081 -name Client1

3rd terminal: 

    go run client/client.go -cPort 8082 -sPort 8081 -name Client2

4th terminal:

    go run client/client.go -cPort 8082 -sPort 8081 -name Client3

Start chatting!

## **Leaving chat**

If you want to leave the chat, simply type 'exit'.