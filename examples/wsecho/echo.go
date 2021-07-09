package main

import (
	"log"
	"net"
	"runtime"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	goeventws "github.com/lemon-mint/go-eventws"
	"github.com/lemon-mint/lemonwork"
)

func main() {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	u := ws.Upgrader{
		OnHeader: func(key, value []byte) (err error) {
			return
		},
	}
	wsserver, err := goeventws.NewServer(32, runtime.NumCPU()*32, runtime.NumCPU()*32*16)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Server Started on localhost:8080")
	wsserver.OnWsMessage = func(data []byte, opcode ws.OpCode, fd int) {
		conn := goeventws.FdRW(fd)
		time.Sleep(time.Second)
		err = wsutil.WriteServerMessage(conn, opcode, data)
		if err != nil {
			log.Println("Error writing server message")
		}
	}

	wsserver.StartPoller()
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("Error on Accept")
			continue
		}
		_, err = u.Upgrade(conn)
		if err != nil {
			log.Println("Failed to upgrade:", conn.RemoteAddr())
			conn.Close()
			continue
		}
		fd := lemonwork.GetFdFromTCPConn(conn.(*net.TCPConn))
		wsserver.AttachClient(fd)
	}
}
