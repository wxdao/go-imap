package imap

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
)

var (
	// ErrResultNo means that NO response is received.
	ErrResultNo = errors.New("result NO")

	// ErrResultBad means that BAD response is received.
	ErrResultBad = errors.New("result BAD")
)

var regexOK, _ = regexp.Compile(`^([0-9]+) ([A-Z]{2,3}).*\r\n$`)
var regexBulkStart, _ = regexp.Compile(`^.*{(\d+)}\r\n$`)

func (c *Client) listen() {
	var line []byte
	var err error
	for line, err = c.r.ReadBytes('\n'); err == nil; line, err = c.r.ReadBytes('\n') {
		if c.Debug != nil {
			fmt.Fprintln(c.Debug, "read: ", string(line))
		}
		switch c.state {
		case justConnected:
			if bytes.HasPrefix(line, []byte("* OK")) {
				c.state = normal
			}
		case normal:
			if bytes.HasPrefix(line, []byte("* BYE")) {
				break
			}
			if bytes.HasPrefix(line, []byte("* ")) {
				if c.selectedMailbox != nil {
					updated := false
					if ind := bytes.Index(line, []byte(" EXISTS\r\n")); ind != -1 {
						c.selectedMailbox.Exists, _ = strconv.Atoi(string(line[2:ind]))
						updated = true
					}
					if ind := bytes.Index(line, []byte(" RECENT\r\n")); ind != -1 {
						c.selectedMailbox.Recent, _ = strconv.Atoi(string(line[2:ind]))
						updated = true
					}
					if updated {
						if c.Debug != nil {
							fmt.Fprintln(c.Debug, "updated: ", *c.selectedMailbox)
						}
						if c.UpdateCallback != nil {
							go c.UpdateCallback()
						}
					}
				}
				c.latestRep = append(c.latestRep, line[2:])
				if submatch := regexBulkStart.FindSubmatch(line); submatch != nil {
					bulkSize, _ := strconv.Atoi(string(submatch[1]))
					bulk := make([]byte, bulkSize)
					_, err := io.ReadFull(c.r, bulk)
					if err != nil {
						break
					}
					_, err = c.r.ReadBytes('\n')
					if err != nil {
						break
					}
					c.latestRep = append(c.latestRep, bulk)
				}
				continue
			}
			if submatch := regexOK.FindSubmatch(line); submatch != nil {
				data := append([][]byte(nil), c.latestRep...)
				c.latestRep = nil
				if c.rep == nil {
					continue
				}
				if string(submatch[2]) == "NO" {
					c.rep <- &repValue{
						err:  ErrResultNo,
						data: nil,
					}
					continue
				}
				if string(submatch[2]) == "BAD" {
					c.rep <- &repValue{
						err:  ErrResultBad,
						data: nil,
					}
					continue
				}
				if c.cmd == "STARTTLS" {
					tlsConn := tls.Client(c.conn, &tls.Config{ServerName: c.tlsHostname})
					err := tlsConn.Handshake()
					if err != nil {
						break
					}
					c.conn = tlsConn
					c.r.Reset(c.conn)
				}
				c.rep <- &repValue{
					err:  nil,
					data: data,
				}
				continue
			}
			c.latestRep = append(c.latestRep, line)
		}
	}
	c.conn = nil
	if c.rep != nil {
		c.rep <- &repValue{
			err:  ErrConnDead,
			data: nil,
		}
	}
}
