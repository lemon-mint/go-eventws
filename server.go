package goeventws

import (
	"syscall"
	"unsafe"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	goworker "github.com/lemon-mint/go-worker"
	"github.com/lemon-mint/lemonwork"
)

type Server struct {
	io lemonwork.NetPoll

	wPool *goworker.Pool

	OnWsMessage func(data []byte, opcode ws.OpCode, fd int)
}

type FdRW int

func (fd FdRW) Read(p []byte) (int, error) {
	return syscall.Read(int(fd), p)
}

func (fd FdRW) Write(p []byte) (int, error) {
	return syscall.Write(int(fd), p)
}

func NewServer(pollerSize int, maxWorkers int, poolsize int) (*Server, error) {
	var err error
	server := &Server{}
	server.io, err = lemonwork.NewPoller(pollerSize)
	if err != nil {
		return nil, err
	}
	server.io.SetAutoClose(true)
	server.wPool = goworker.NewPool(maxWorkers, poolsize)
	server.io.SetOnDataCallback(func(fd int) {
		server.wPool.RunTask(unsafe.Pointer(&fd))
	})
	server.wPool.HandlerFunc = server.onDataCallback
	return server, nil
}

func (server *Server) AttachClient(fd int) error {

	err := syscall.SetNonblock(fd, true)
	if err != nil {
		return err
	}

	err = server.io.Register(fd)
	return err
}

func (server *Server) onDataCallback(task goworker.Task) {
	fd := *(*int)(task.Data)
	payload, opcode, err := wsutil.ReadClientData(FdRW(fd))
	if err != nil {
		return
	}
	if opcode == ws.OpClose {
		syscall.Close(fd)
	}
	if server.OnWsMessage != nil {
		server.OnWsMessage(payload, opcode, fd)
	}
}

func (server *Server) SetOncloseCallback(f func(fd int)) {
	server.io.SetOnCloseCallback(f)
}

func (server *Server) Poll() error {
	return server.io.Poll()
}
