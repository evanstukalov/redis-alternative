package main

import (
	"flag"
	"net"
	"os"

	nested "github.com/antonfisher/nested-logrus-formatter"
	log "github.com/sirupsen/logrus"

	"github.com/codecrafters-io/redis-starter-go/internal/config"
	"github.com/codecrafters-io/redis-starter-go/internal/master"
	"github.com/codecrafters-io/redis-starter-go/internal/redis"
	"github.com/codecrafters-io/redis-starter-go/internal/slave"
)

func init() {
	f := &nested.Formatter{}
	log.SetFormatter(f)

	log.SetOutput(os.Stdout)

	log.SetLevel(log.DebugLevel)
}

func parseFlags() config.Flags {
	port := flag.Int("port", 6379, "Port to listen on")
	replicaOf := flag.String("replicaof", "", "Replica to another server")
	dir := flag.String("dir", "", "Directory to store data")
	dbFileName := flag.String("dbfilename", "", "Database file name")

	flag.Parse()
	return config.Flags{
		Port:       *port,
		ReplicaOf:  *replicaOf,
		Dir:        *dir,
		DbFileName: *dbFileName,
	}
}

func main() {
	connChan := make(chan net.Conn)
	errChan := make(chan error)

	flags := parseFlags()
	services := config.InitializeServices()
	ctx := config.CreateContext(services)

	config := config.NewConfig()
	config.Initialize(&flags)

	listener := master.CreateListener(config)
	defer listener.Close()

	redis.LoadRDB(ctx, config.GetRedisDir(), config.GetRedisDbFileName())

	if config.GetRole() == "slave" {
		slave.HandleSlaveMode(ctx, config)
	}

	go master.AcceptConnections(listener, connChan, errChan)
	go services.ExpiredCollectorObj.Tick()

	master.HandleConnections(ctx, connChan, errChan, config)
}
