package clients

import (
	"fmt"
	"net"
	"sync"
)

type Clients struct {
	Clients map[net.Conn]struct{}
	Mutex   sync.RWMutex
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

func (cl *Clients) Get() []net.Conn {
	cl.Mutex.RLock()
	defer cl.Mutex.RUnlock()
	keys := make([]net.Conn, 0, len(cl.Clients))

	fmt.Printf("Found %d connected clients\r\n", len(cl.Clients))

	for key := range cl.Clients {
		keys = append(keys, key)
	}

	return keys
}
