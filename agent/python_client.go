package agent

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

type PythonClient struct {
	conn   net.Conn
	reader *bufio.Reader
}

// Strips the port from host if it's present, e.g. "192.168.0.9:8080" => "192.168.0.9"
func stripPortIfNeeded(host string) string {
	if strings.Contains(host, ":") {
		strippedHost, _, err := net.SplitHostPort(host)
		if err == nil {
			return strippedHost
		}
	}
	return host
}

func NewPythonClient(host string, port int) (*PythonClient, error) {
	cleanHost := stripPortIfNeeded(host)
	fullAddr := fmt.Sprintf("%s:%d", cleanHost, port)

	conn, err := net.DialTimeout("tcp", fullAddr, 5*time.Second)
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
