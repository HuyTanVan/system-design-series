package server

import (
	"fmt"
	"redis-clone/command"
	"redis-clone/resp"
	"net"
)

func handleConn(conn net.Conn, d *command.Dispatcher) {
	// close the connection when the server is stopped
	defer conn.Close()

	for {
		// accept a new connection
		r := resp.NewResp(conn)
		// read request
		value, err := r.Read()
		if err != nil {
			fmt.Println("read error:", err)
			return
		}
		// dispatch command and receive response
		respValue := d.Dispatch(value)

		w := resp.NewWriter(conn)
		err = w.Write(respValue)
		if err != nil {
			fmt.Println("write error:", err)
			return
		}
		// // echo back a hardcoded OK for now
		// conn.Write([]byte("+OK\r\n"))
	}
}
