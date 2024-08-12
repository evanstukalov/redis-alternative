package clients

import (
	"fmt"
	"net"
	"sync"
)

type Clients struct {
	Clients map[net.Conn]struct{}
	Mutex   sync.Mutex
}

func NewClients() *Clients {
	return &Clients{
		Clients: make(map[net.Conn]struct{}),
	}
}

func (cl *Clients) Set(client net.Conn) {
	cl.Mutex.Lock()
	defer cl.Mutex.Unlock()

	fmt.Println("New client has been connected! ", client.RemoteAddr().String())

	cl.Clients[client] = struct{}{}
}
