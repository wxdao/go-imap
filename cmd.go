package imap

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
)

var (
	// ErrNilRep means that nil response is received.
	ErrNilRep = errors.New("nil response")
)

var regexFetchResult, _ = regexp.Compile(`^(\d+) FETCH.*\r\n$`)

func (c *Client) newTag() string {
	return strconv.Itoa(int(atomic.AddInt32(&c.latestTag, 1)))
}

func (c *Client) prepareCmd(cmd string) (tag string, err error) {
	c.cmd = cmd
	if c.conn == nil {
		return
	}
	tag = c.newTag()
	c.mux.Lock()
	c.rep = make(chan *repValue, 1)
	return
}

func (c *Client) cleanCmd() {
	c.rep = nil
	c.mux.Unlock()
	return
}

// StartTLS performs STARTTLS command.
func (c *Client) StartTLS(hostname string) (err error) {
	tag, err := c.prepareCmd("STARTTLS")
	if err != nil {
		return
	}
	defer c.cleanCmd()

	c.tlsHostname = hostname

	err = c.writeString(tag + " STARTTLS\r\n")
	if err != nil {
		return
	}

	rep := <-c.rep
	if rep == nil {
		err = ErrNilRep
		return
	}
	if rep.err != nil {
		err = rep.err
		return
	}
	return
}

// Capability performs CAPIBILITY command.
func (c *Client) Capability() (caps []string, err error) {
	tag, err := c.prepareCmd("CAPABILITY")
	if err != nil {
		return
	}
	defer c.cleanCmd()

	err = c.writeString(tag + " CAPABILITY\r\n")
	if err != nil {
		return
	}

	rep := <-c.rep
	if rep == nil {
		err = ErrNilRep
		return
	}
	if rep.err != nil {
		err = rep.err
		return
	}
	for _, line := range rep.data {
		lineCaps := strings.Split(string(line[:len(line)-2]), " ")
		if lineCaps[0] != "CAPABILITY" {
			continue
		}
		caps = append(caps, lineCaps[1:]...)
	}
	return
}

// Noop performs NOOP command.
func (c *Client) Noop() (err error) {
	tag, err := c.prepareCmd("NOOP")
	if err != nil {
		return
	}
	defer c.cleanCmd()

	err = c.writeString(tag + " NOOP\r\n")
	if err != nil {
		return
	}

	rep := <-c.rep
	if rep == nil {
		err = ErrNilRep
		return
	}
	if rep.err != nil {
		err = rep.err
		return
	}
	return
}

// Logout performs LOGOUT command.
func (c *Client) Logout() (err error) {
	tag, err := c.prepareCmd("LOGOUT")
	if err != nil {
		return
	}
	defer c.cleanCmd()

	err = c.writeString(tag + " LOGOUT\r\n")
	if err != nil {
		return
	}

	rep := <-c.rep
	if rep == nil {
		err = ErrNilRep
		return
	}
	if rep.err != nil {
		err = rep.err
		return
	}
	return
}

// Login performs LOGIN command.
func (c *Client) Login(user string, pass string) (err error) {
	tag, err := c.prepareCmd("LOGIN")
	if err != nil {
		return
	}
	defer c.cleanCmd()

	err = c.writeString(fmt.Sprintf("%s %s %s %s\r\n", tag, "LOGIN", user, pass))
	if err != nil {
		return
	}

	rep := <-c.rep
	if rep == nil {
		err = ErrNilRep
		return
	}
	if rep.err != nil {
		err = rep.err
		return
	}
	return
}

// Select performs SELECT command.
func (c *Client) Select(name string) (info *MailboxInfo, err error) {
	tag, err := c.prepareCmd("SELECT")
	if err != nil {
		return
	}
	defer c.cleanCmd()

	err = c.writeString(tag + " SELECT " + name + "\r\n")
	if err != nil {
		return
	}

	rep := <-c.rep
	if rep == nil {
		err = ErrNilRep
		return
	}
	if rep.err != nil {
		err = rep.err
		return
	}
	info = &MailboxInfo{Name: name}
	c.selectedMailbox = info
	for _, line := range rep.data {
		lineString := string(line)
		if ind := strings.Index(lineString, " EXISTS\r\n"); ind != -1 {
			info.Exists, _ = strconv.Atoi(lineString[:ind])
			continue
		}
		if ind := strings.Index(lineString, " RECENT\r\n"); ind != -1 {
			info.Recent, _ = strconv.Atoi(lineString[:ind])
			continue
		}
		if strings.HasPrefix(lineString, "FLAGS ") {
			flagsString := lineString[6 : len(line)-2]
			flagsString = strings.Replace(flagsString, "(", "", 1)
			flagsString = strings.Replace(flagsString, ")", "", 1)
			info.Flags = strings.Split(flagsString, " ")
			continue
		}
	}
	return
}

// Search performs SEARCH command.
func (c *Client) Search(criteria string) (seqs []int, err error) {
	tag, err := c.prepareCmd("SEARCH")
	if err != nil {
		return
	}
	defer c.cleanCmd()

	err = c.writeString(tag + " SEARCH " + criteria + "\r\n")
	if err != nil {
		return
	}

	rep := <-c.rep
	if rep == nil {
		err = ErrNilRep
		return
	}
	if rep.err != nil {
		err = rep.err
		return
	}
	for _, line := range rep.data {
		lineSeqs := strings.Split(string(line[:len(line)-2]), " ")
		if lineSeqs[0] != "SEARCH" {
			continue
		}
		for _, seqStr := range lineSeqs[1:] {
			seq, _ := strconv.Atoi(seqStr)
			seqs = append(seqs, seq)
		}
	}
	return
}

// FetchRFC822 performs FETCH command to fetch RFC822 data.
func (c *Client) FetchRFC822(seqs []int) (data map[int][]byte, err error) {
	tag, err := c.prepareCmd("FETCH")
	if err != nil {
		return
	}
	defer c.cleanCmd()

	var strSeqs []string
	for _, seq := range seqs {
		strSeqs = append(strSeqs, strconv.Itoa(seq))
	}

	err = c.writeString(tag + " FETCH " + strings.Join(strSeqs, ",") + " RFC822\r\n")
	if err != nil {
		return
	}

	rep := <-c.rep
	if rep == nil {
		err = ErrNilRep
		return
	}
	if rep.err != nil {
		err = rep.err
		return
	}
	data = make(map[int][]byte)
	for i := 0; i < len(rep.data); {
		line := rep.data[i]
		if submatch := regexFetchResult.FindSubmatch(line); submatch != nil {
			seq, _ := strconv.Atoi(string(submatch[1]))
			data[seq] = rep.data[i+1]
			i += 3
			continue
		}
		i++
	}
	return
}

// FetchRFC822Header performs FETCH command to fetch RFC822.HEADER data.
func (c *Client) FetchRFC822Header(seqs []int) (data map[int][]byte, err error) {
	tag, err := c.prepareCmd("FETCH")
	if err != nil {
		return
	}
	defer c.cleanCmd()

	var strSeqs []string
	for _, seq := range seqs {
		strSeqs = append(strSeqs, strconv.Itoa(seq))
	}

	err = c.writeString(tag + " FETCH " + strings.Join(strSeqs, ",") + " RFC822.HEADER\r\n")
	if err != nil {
		return
	}

	rep := <-c.rep
	if rep == nil {
		err = ErrNilRep
		return
	}
	if rep.err != nil {
		err = rep.err
		return
	}
	data = make(map[int][]byte)
	for i := 0; i < len(rep.data); {
		line := rep.data[i]
		if submatch := regexFetchResult.FindSubmatch(line); submatch != nil {
			seq, _ := strconv.Atoi(string(submatch[1]))
			data[seq] = rep.data[i+1]
			i += 3
			continue
		}
		i++
	}
	return
}

// Idle performs IDLE command.
func (c *Client) Idle() (err error) {
	tag, err := c.prepareCmd("IDLE")
	if err != nil {
		return
	}
	defer c.cleanCmd()

	err = c.writeString(tag + " IDLE\r\n")
	if err != nil {
		return
	}

	c.doneEmitted = false
	c.idling = true
	defer func() {
		c.idling = false
		c.doneEmitted = false
	}()

	rep := <-c.rep
	if rep == nil {
		err = ErrNilRep
		return
	}
	if rep.err != nil {
		err = rep.err
		return
	}
	return
}

// Done performs DONE command.
func (c *Client) Done() (err error) {
	c.doneMux.Lock()
	defer c.doneMux.Unlock()
	if c.idling && !c.doneEmitted {
		err = c.writeString("DONE\r\n")
		c.doneEmitted = true
	}
	return
}
