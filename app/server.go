package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
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
	reader := bufio.NewReader(conn)

	for {
		// Parse the RESP command
		command, args, err := parseRESP(reader)
		if err != nil {
			fmt.Println("Error Reading:", err.Error())
			return
		}

		// Handle the PING command
		if strings.ToUpper(command) == "PING" {
			conn.Write([]byte("+PONG\r\n"))
		} else if strings.ToUpper(command) == "ECHO" && len(args) == 1 {
			// Handle the ECHO command
			response := fmt.Sprintf("$%d\r\n%s\r\n", len(args[0]), args[0])
			conn.Write([]byte(response))
		} else {
			conn.Write([]byte("-ERR unknown command\r\n"))
		}
	}
}

func parseRESP(reader *bufio.Reader) (string, []string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", nil, err
	}

	if line[0] != '*' {
		return "", nil, fmt.Errorf("expected array")
	}

	numArgs := 0
	fmt.Sscanf(line, "*%d\r\n", &numArgs)

	args := make([]string, numArgs)
	for i := 0; i < numArgs; i++ {
		// Read the bulk string header
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", nil, err
		}

		if line[0] != '$' {
			return "", nil, fmt.Errorf("expected bulk string")
		}

		argLen := 0
		fmt.Sscanf(line, "$%d\r\n", &argLen)

		// Read the argument
		arg := make([]byte, argLen)
		_, err = reader.Read(arg)
		if err != nil {
			return "", nil, err
		}

		// Read the trailing \r\n
		_, err = reader.ReadString('\n')
		if err != nil {
			return "", nil, err
		}

		args[i] = string(arg)
	}

	return args[0], args[1:], nil
}
