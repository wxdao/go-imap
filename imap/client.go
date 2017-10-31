package imap

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

var (
	// ErrConnDead means that conn is dead.
	ErrConnDead = errors.New("conn dead")
)

const (
	justConnected = iota
	normal
)

type repValue struct {
	err  error
	data [][]byte
}

// Client is an IMAP client.
type Client struct {
	conn            net.Conn
	r               *bufio.Reader
	state           int
	latestTag       int32
	latestRep       [][]byte
	bulkSize        int
	mux             sync.Mutex
	doneMux         sync.Mutex
	doneEmitted     bool
	rep             chan *repValue
	cmd             string
	tlsHostname     string
	selectedMailbox *MailboxInfo
	idling          bool
	UpdateCallback  func()
	Debug           io.Writer
}

func (c *Client) write(data []byte) (err error) {
	if c.Debug != nil {
		fmt.Fprintln(c.Debug, "write: ", string(data))
	}
	_, err = c.conn.Write(data)
	if err != nil {
		err = ErrConnDead
	}
	return
}

func (c *Client) writeString(data string) error {
	return c.write([]byte(data))
}

// GetSelectedMailboxInfo gets current selected mailbox's info.
func (c *Client) GetSelectedMailboxInfo() (info MailboxInfo) {
	if c.selectedMailbox != nil {
		info = *c.selectedMailbox
	}
	return
}

// Dial connects to an IMAP server.
func Dial(addr string) (c *Client, err error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	c = &Client{
		conn: conn,
		r:    bufio.NewReader(conn),
	}
	go c.listen()
	return
}

// DialTLS connects to an IMAP server over TLS.
func DialTLS(addr string, tlsConfig *tls.Config) (c *Client, err error) {
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return nil, err
	}
	c = &Client{
		conn: conn,
		r:    bufio.NewReader(conn),
	}
	go c.listen()
	return
}
