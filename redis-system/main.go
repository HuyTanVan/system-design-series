package main

import (
	"log"
	"redis-clone/server"
)

func main() {
	const ADDR = ":6379"
	// 1. create a new TCP listener on port 6379
	server := server.NewServer(ADDR)
	log.Fatal(server.Start())
	// // 1. create a new TCP listener on port 6379
	// l, err := net.Listen("tcp", ":6379")
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// // 2. create a new AOF for persistence
	// aof, err := NewAof("database.aof")
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// // close the AOF when the server is stopped
	// defer aof.Close()

	// // 3. listen for connections
	// conn, err := l.Accept()
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// // close the connection when the server is stopped
	// defer conn.Close()

	// // 4. handle the connection, events happen in a loop until the connection is closed
	// for {
	// 	resp := NewResp(conn)
	// 	value, err := resp.Read()
	// 	if err != nil {
	// 		fmt.Println(err)
	// 		return
	// 	}

	// 	if value.typ != "array" {
	// 		fmt.Println("Invalid request, expected array")
	// 		continue
	// 	}

	// 	if len(value.array) == 0 {
	// 		fmt.Println("Invalid request, expected array length > 0")
	// 		continue
	// 	}

	// 	command := strings.ToUpper(value.array[0].bulk)
	// 	args := value.array[1:]

	// 	writer := NewWriter(conn)

	// 	handler, ok := Handlers[command]
	// 	if !ok {
	// 		fmt.Println("Invalid command: ", command)
	// 		writer.Write(Value{typ: "string", str: ""})
	// 		continue
	// 	}

	// 	if command == "SET" || command == "HSET" {
	// 		aof.Write(value)
	// 	}

	// 	result := handler(args)
	// 	writer.Write(result)
	// }
}

// package main

// import (
// 	"bytes"
// )

// func main() {
// 	input := "*3\r\n$3\r\nSET\r\n$4\r\nname\r\n$3\r\nhuy\r\n"
// 	resp := NewResp(bytes.NewReader([]byte(input)))

// 	for {
// 		_, _, err := resp.readLine()
// 		if err != nil {
// 			break
// 		}
// 		// fmt.Printf("Read %d bytes: %s\n", n, line)
// 	}
// }
