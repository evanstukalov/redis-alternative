package clients

import (
	"net"
	"sync"
)

type offset int64

type Clients struct {
	Clients    map[net.Conn]offset
	Mutex      sync.RWMutex
	Subscriber func(conn net.Conn, offset int)
}

func NewClients() *Clients {
	return &Clients{
		Clients: make(map[net.Conn]offset),
	}
}

func (cl *Clients) Set(client net.Conn) {
	cl.Mutex.Lock()
	defer cl.Mutex.Unlock()

	cl.Clients[client] = 0
}

func (cl *Clients) GetAll() []net.Conn {
	cl.Mutex.RLock()
	defer cl.Mutex.RUnlock()
	keys := make([]net.Conn, 0, len(cl.Clients))

	for key := range cl.Clients {
		keys = append(keys, key)
	}

	return keys
}

func (cl *Clients) SetOffset(conn net.Conn, n int) {
	cl.Mutex.Lock()
	defer cl.Mutex.Unlock()

	cl.Clients[conn] = offset(n)

	cl.Notify(conn, n)
}

func (cl *Clients) GetOffset(conn net.Conn) offset {
	cl.Mutex.Lock()
	defer cl.Mutex.Unlock()
	value := cl.Clients[conn]

	return value
}

func (cl *Clients) Notify(conn net.Conn, offset int) {
	if cl.Subscriber != nil {
		cl.Subscriber(conn, offset)
	}
}

func (cl *Clients) NotifyAll(offset int) {
	cl.Mutex.RLock()
	defer cl.Mutex.RLock()

	for _, client := range cl.GetAll() {
		cl.Notify(client, offset)
	}
}

func (cl *Clients) Subscribe(handler func(conn net.Conn, clientOffset int)) {
	cl.Mutex.Lock()
	defer cl.Mutex.Unlock()

	cl.Subscriber = handler
}
