package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

// var sslRequest = []byte{0x00, 0x00, 0x00, 0x08, 0x04, 0xd2, 0x16, 0x2f}
var authenticationOk = []byte{0x52, 0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00}

// 0x49 means Idle Status
var readyForQuery = []byte{0x5a, 0x00, 0x00, 0x00, 0x05, 0x49}

// ref: https://www.postgresql.org/docs/current/protocol-message-formats.html#PROTOCOL-MESSAGE-FORMATS-COMMANDCOMPLETE
// []byte{0x43} means CommandComplete type
// []byte{0x4f, 0x4b} means OK
var acceptMsg = []byte{0x43, 0x00, 0x00, 0x00, 0x7, 0x4f, 0x4b, 0x00}

type TCPServerConfig struct {
	Port         int
	QueryHandler func(qry []byte) ([]byte, error)
}

type TCPServer struct {
	port       int
	qryHandler func(qry []byte) ([]byte, error)

	// additional parameters
	serverVersion string
	timezone      string
}

func NewTCPServer(conf *TCPServerConfig, opts ...tcpServerOption) *TCPServer {
	s := &TCPServer{
		port:          conf.Port,
		qryHandler:    conf.QueryHandler,
		serverVersion: "0.0",
		timezone:      "UTC",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *TCPServer) RegisterHandler(handler func(qry []byte) ([]byte, error)) {
	s.qryHandler = handler
}

func (s *TCPServer) Start() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}
	defer ln.Close()

	if s.qryHandler == nil {
		return fmt.Errorf("query handler is not registered")
	}

	log.Println("Server started")

	for {
		conn, err := ln.Accept()
		log.Printf("New connection from: %s", conn.RemoteAddr())

		if err != nil {
			log.Printf("Error accepting connection: %+v", err)
			return err
		}
		go s.handleConnection(conn)
	}
}

func (s *TCPServer) handleConnection(c net.Conn) {
	_ = s.startup(c)
	defer c.Close()

	for {
		typ, msg, err := s.readMessage(c)
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Printf("Connection closed by client %s", c.RemoteAddr())
				break
			} else {
				log.Printf("Error reading message: %+v", err)
				os.Exit(1)
			}
		}
		if typ[0] == 0x58 {
			log.Printf("Received Terminate message from %s", c.RemoteAddr())
			return // Terminate connection
		}

		if typ[0] == 0x51 {
			// Simple query
			res, err := s.qryHandler(msg)
			if err != nil {
				// TODO: handle error
			}

			if res == nil {
				c.Write(acceptMsg)
			} else {
				c.Write(s.buildCompletedResponse(res))
			}

			c.Write(readyForQuery)
		}
	}
}

func (s *TCPServer) buildParameters() []byte {
	p := make([]byte, 0)
	p = append(p, s.buildParameterStatus("TimeZone", s.timezone)...)
	p = append(p, s.buildParameterStatus("server_version", s.serverVersion)...)

	return p
}

func (s *TCPServer) buildParameterStatus(name, value string) []byte {
	// ref: https://www.postgresql.org/docs/current/protocol-message-formats.html#PROTOCOL-MESSAGE-FORMATS-PARAMETERSTATUS
	key := []byte(name)
	key = append(key, 0x00) // null

	val := []byte(value)
	val = append(val, 0x00) // null

	length := 4 + len(key) + len(val)
	len := make([]byte, 4)
	binary.BigEndian.PutUint32(len, uint32(length))

	p := make([]byte, 0, length+1)
	p = append(p, 0x53) // ParameterStatus type
	p = append(p, len...)
	p = append(p, key...)
	p = append(p, val...)

	return p
}

func (s *TCPServer) buildCompletedResponse(body []byte) []byte {
	body = append(body, 0x00)
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(body)+4))

	payload := make([]byte, 0, len(body)+len(length)+1)
	payload = append(payload, 0x43) // CommandComplete type
	payload = append(payload, length...)
	payload = append(payload, body...)
	return payload
}

// ref: https://www.postgresql.org/docs/current/protocol-message-formats.html#PROTOCOL-MESSAGE-FORMATS
func (s *TCPServer) readMessage(c net.Conn) (typ, message []byte, err error) {
	typ, err = s.read(c, 1)
	if err != nil {
		return nil, nil, err
	}
	len, err := s.read(c, 4)
	if err != nil {
		return nil, nil, err
	}
	message, err = s.read(c, int(binary.BigEndian.Uint32(len))-4)
	if err != nil {
		return nil, nil, err
	}

	return typ, message, nil
}

// ref: https://www.postgresql.org/docs/current/protocol-message-formats.html#PROTOCOL-MESSAGE-FORMATS-QUERY

func (s *TCPServer) startup(c net.Conn) error {
	sizeByte, err := s.read(c, 4)
	if err != nil {
		return err
	}
	// TODO: support SSL
	c.Write([]byte{0x4e}) // N means not support SSL

	size := int(binary.BigEndian.Uint32(sizeByte))
	// consider offset 4 bytes
	if _, err := s.read(c, size-4); err != nil {
		return err
	}

	// AuthenticationOk
	c.Write(authenticationOk)

	// ref: https://www.postgresql.org/docs/current/protocol-message-formats.html#PROTOCOL-MESSAGE-FORMATS-PARAMETERSTATUS
	c.Write(s.buildParameters())

	// ReadyForQuery
	c.Write(readyForQuery)

	return nil
}

func (s *TCPServer) read(r io.Reader, len int) ([]byte, error) {
	buf := make([]byte, len)
	n, err := r.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("error reading from connection: %w", err)
	}
	return buf[:n], nil
}
