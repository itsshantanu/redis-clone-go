package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	store      = make(map[string]string)
	expiry     = make(map[string]*time.Timer)
	expiryLock sync.Mutex
	dir        string
	dbfilename string
)

func parseArgs() {
	for i := 0; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--dir":
			if i+1 < len(os.Args) {
				dir = os.Args[i+1]
				i++
			}
		case "--dbfilename":
			if i+1 < len(os.Args) {
				dbfilename = os.Args[i+1]
				i++
			}
		}
	}
}

func main() {
	parseArgs()
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

		switch strings.ToUpper(command) {
		case "PING":
			conn.Write([]byte("+PONG\r\n"))
		case "ECHO":
			if len(args) == 1 {
				response := fmt.Sprintf("$%d\r\n%s\r\n", len(args[0]), args[0])
				conn.Write([]byte(response))
			} else {
				conn.Write([]byte("-ERR wrong number of arguments for 'echo' command\r\n"))
			}
		case "SET":
			if len(args) >= 2 {
				key := args[0]
				value := args[1]
				store[key] = value
				conn.Write([]byte("+OK\r\n"))

				if len(args) == 4 && strings.ToUpper(args[2]) == "PX" {
					expiryTime, err := time.ParseDuration(args[3] + "ms")
					if err == nil {
						expiryLock.Lock()
						if timer, exists := expiry[key]; exists {
							timer.Stop()
						}
						expiry[key] = time.AfterFunc(expiryTime, func() {
							expiryLock.Lock()
							delete(store, key)
							delete(expiry, key)
							expiryLock.Unlock()
						})
						expiryLock.Unlock()
					}
				}
			} else {
				conn.Write([]byte("-ERR wrong number of arguments for 'set' command\r\n"))
			}
		case "GET":
			if len(args) == 1 {
				key := args[0]
				expiryLock.Lock()
				value, exists := store[key]
				expiryLock.Unlock()
				if exists {
					response := fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)
					conn.Write([]byte(response))
				} else {
					conn.Write([]byte("$-1\r\n"))
				}
			} else {
				conn.Write([]byte("-ERR wrong number of arguments for 'get' command\r\n"))
			}
		case "CONFIG":
			if len(args) == 2 && strings.ToUpper(args[0]) == "GET" {
				param := args[1]
				var value string
				switch param {
				case "dir":
					value = dir
				case "dbfilename":
					value = dbfilename
				default:
					conn.Write([]byte("-ERR unknown parameter\r\n"))
					continue
				}
				response := fmt.Sprintf("*2\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(param), param, len(value), value)
				conn.Write([]byte(response))
			} else {
				conn.Write([]byte("-ERR wrong number of arguments for 'CONFIG GET' command\r\n"))
			}
		default:
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
