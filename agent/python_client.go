package agent

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

type PythonClient struct {
	conn   net.Conn
	reader *bufio.Reader
}

func NewPythonClient(host string, port int) (*PythonClient, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	if err != nil {
		return nil, err
	}
	return &PythonClient{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}, nil
}

func (p *PythonClient) Exec(code string) (string, error) {
	_, err := p.conn.Write([]byte(code + "\n"))
	if err != nil {
		return "", err
	}
	resp, err := p.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return resp, nil
}
