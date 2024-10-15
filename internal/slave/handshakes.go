package slave

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/interfaces"
)

type MasterInfo struct {
	Host string
	Port string
}

func (m MasterInfo) Address() string {
	return fmt.Sprintf("%s:%s", m.Host, m.Port)
}

func masterInfoFromParam(replicaOf string) MasterInfo {
	var delim string

	if strings.Contains(replicaOf, ":") {
		delim = ":"
	} else if strings.Contains(replicaOf, " ") {
		delim = " "
	} else {
		log.Fatalln("Error parsing masterInfo")
	}

	data := strings.Split(replicaOf, delim)
	return MasterInfo{
		Host: data[0],
		Port: data[1],
	}
}

func sendMessage(conn net.Conn, message string) error {
	if _, err := conn.Write([]byte(message)); err != nil {
		return err
	}
	return nil
}

func readAnswer(
	conn net.Conn,
) {
	_, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println("Error reading from connection: ", err.Error())
		return
	}
}

func readNBytes(reader io.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(reader, buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func ConnectMaster(config interfaces.IConfig) (net.Conn, error) {
	masterInfo := masterInfoFromParam(config.GetSlave().GetReplicaOf())
	addr := masterInfo.Address()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println("Error connecting to master: ", err)
		return nil, err
	}
	return conn, nil
}

func Handshakes(conn net.Conn, config interfaces.IConfig) (*bufio.Reader, error) {
	if err := sendMessage(conn, "*1\r\n$4\r\nPING\r\n"); err != nil {
		return nil, err
	}
	readAnswer(conn)
	if err := sendMessage(
		conn,
		fmt.Sprintf("*3\r\n$8\r\nREPLCONF\r\n$14\r\nlistening-port\r\n$4\r\n%d\r\n", config.GetPort()),
	); err != nil {
		return nil, err
	}
	readAnswer(conn)
	if err := sendMessage(conn, "*5\r\n$8\r\nREPLCONF\r\n$4\r\ncapa\r\n$3\r\neof\r\n$4\r\ncapa\r\n$6\r\npsync2\r\n"); err != nil {
		return nil, err
	}
	readAnswer(conn)
	if err := sendMessage(conn, "*3\r\n$5\r\nPSYNC\r\n$1\r\n?\r\n$2\r\n-1\r\n"); err != nil {
		return nil, err
	}
	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	line, err = reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	var dataLen int
	_, err = fmt.Sscanf(string(line), "$%d\r\n", &dataLen)
	if err != nil {
		return nil, err
	}

	_, err = readNBytes(reader, dataLen)
	if err != nil {
		return nil, err
	}

	// fmt.Println("Received RDB:", string(data))

	return reader, nil
}
