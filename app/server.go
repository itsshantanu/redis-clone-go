package main

import (
	"fmt"
	//Uncomment this block to pass the first stage
	"net"
	"os"
)

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	//Start listening on the specified port
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	defer l.Close()

	for {
		//Accept a new connection
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			continue
		}
		//Handle the connection in a new goroutine
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)

	for {
		//Read data from the connection
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error Reading:", err.Error())
			return
		}
		// Process the command
		command := string(buf[:n])
		fmt.Printf("Received Command: %s", command)
		conn.Write([]byte("+PONG\r\n"))
	}
}
