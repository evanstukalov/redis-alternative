package clients

import (
	"net"
	"sync"

	"github.com/sirupsen/logrus"
)

type offset int64

type Clients struct {
	Clients    map[net.Conn]offset
	Mutex      sync.RWMutex
	Subscriber func(conn net.Conn, offset int)
}

func NewClients() *Clients {
	logrus.Info("Creating new clients")
	return &Clients{
		Clients: make(map[net.Conn]offset),
	}
}

func (cl *Clients) Set(client net.Conn) {
	cl.Mutex.Lock()
	defer cl.Mutex.Unlock()

	logrus.WithFields(logrus.Fields{
		"package":  "clients",
		"function": "Set",
		"Address":  client.RemoteAddr().String(),
	}).Info("New client has been connected")

	cl.Clients[client] = 0
}

func (cl *Clients) GetAll() []net.Conn {
	cl.Mutex.RLock()
	defer cl.Mutex.RUnlock()
	keys := make([]net.Conn, 0, len(cl.Clients))

	logrus.WithFields(logrus.Fields{
		"package":  "clients",
		"function": "GetAll",
		"value":    cl.Clients,
	}).Info("Getting all clients")

	for key := range cl.Clients {
		keys = append(keys, key)
	}

	return keys
}

func (cl *Clients) SetOffset(conn net.Conn, n int) {
	cl.Mutex.Lock()
	defer cl.Mutex.Unlock()

	logrus.WithFields(logrus.Fields{
		"package":  "clients",
		"function": "SetOffset",
		"value":    offset(n),
	}).Info("Changing offset of client")

	cl.Clients[conn] = offset(n)

	cl.Notify(conn, n)
}

func (cl *Clients) GetOffset(conn net.Conn) offset {
	cl.Mutex.Lock()
	defer cl.Mutex.Unlock()
	value := cl.Clients[conn]

	logrus.WithFields(logrus.Fields{
		"package":  "clients",
		"function": "GetOffset",

		"value": value,
	}).Info("Getting offset of client")

	return value
}

func (cl *Clients) Notify(conn net.Conn, offset int) {
	logrus.WithFields(logrus.Fields{
		"package":  "clients",
		"function": "Notify",
	}).Info("Notify subsriber")

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

	logrus.WithFields(logrus.Fields{
		"package":  "clients",
		"function": "Subscribe",
	}).Info("New handler subsribed.")

	cl.Subscriber = handler
}
