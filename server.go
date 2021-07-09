package goeventws

import (
	"sync/atomic"
	"syscall"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/lemon-mint/lemonwork"
	"github.com/lemon-mint/unlock"
)

type Server struct {
	evLoops []eventLoop
	numEvs  int

	OnWsMessage func(data []byte, opcode ws.OpCode, fd int)
	onClose     func(fd int)
}

type eventLoop struct {
	io     lemonwork.NetPoll
	buffer *unlock.RingBuffer

	connection int64
}

func (e *eventLoop) worker(hf func(data []byte, opcode ws.OpCode, fd int), onClose func(fd int)) {
	e.io.SetOnDataCallback(
		func(fd int) {
			payload, opcode, err := wsutil.ReadClientData(FdRW(fd))
			if err != nil {
				return
			}
			if opcode == ws.OpClose {
				syscall.Close(fd)
			}
			hf(payload, opcode, fd)
		},
	)
	e.io.SetOnCloseCallback(
		func(fd int) {
			atomic.AddInt64(&e.connection, -1)
			onClose(fd)
		},
	)
	e.io.SetAutoClose(true)
	for {
		e.io.Poll()
	}
}

type FdRW int

func (fd FdRW) Read(p []byte) (int, error) {
	return syscall.Read(int(fd), p)
}

func (fd FdRW) Write(p []byte) (int, error) {
	return syscall.Write(int(fd), p)
}

func NewServer(pollerSize int, loops int, poolsize int) (*Server, error) {
	var err error
	server := &Server{}
	server.evLoops = make([]eventLoop, loops)
	for i := range server.evLoops {
		server.evLoops[i].io, err = lemonwork.NewPoller(pollerSize)
		if err != nil {
			return nil, err
		}
		server.evLoops[i].buffer = unlock.NewRingBuffer(poolsize)
	}
	server.numEvs = loops

	return server, nil
}

func (server *Server) AttachClient(fd int) error {

	err := syscall.SetNonblock(fd, true)
	if err != nil {
		return err
	}
	atomic.AddInt64(&server.evLoops[fd%server.numEvs].connection, 1)
	err = server.evLoops[fd%server.numEvs].io.Register(fd)
	return err
}

func (server *Server) StartPoller() {
	for i := range server.evLoops {
		go server.evLoops[i].worker(server.OnWsMessage, server.onClose)
	}
}

func (server *Server) SetOncloseCallback(f func(fd int)) {
	server.onClose = f
}
