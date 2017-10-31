# go-imap

An IMAP client library in Go.

So far it only implements a subset of IMAP commands, but it's been enough for simple email retrieving jobs.

**IDLE and status change callback are supported.**

[GoDoc](https://godoc.org/github.com/wxdao/go-imap/imap)

## Example

```go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/wxdao/go-imap/imap"
)

func main() {
	client, err := imap.Dial("imap.mail.com:143")
	if err != nil {
		panic(err)
	}

	interrupted := make(chan os.Signal, 1)
	signal.Notify(interrupted, os.Interrupt, os.Kill)

	updated := make(chan int)

  	// invoked when status changed
	client.UpdateCallback = func() {
		updated <- 1
	}

	client.StartTLS("imap.mail.com")
	client.Login("bot@mail.com", "I'm a mail bot.")
	client.Select("INBOX")

loop:
	for {
		seqs, err := client.Search("UNSEEN")
		if err != nil {
			panic(err)
		}
		if len(seqs) > 0 {
			data, err := client.FetchRFC822(seqs)
			if err != nil {
				panic(err)
			}
			go handleNewEmails(data)
		}
		go client.Idle()
		select {
		case <-updated:
			client.Done()
		case <-time.After(time.Minute * 10):
			client.Done()
		case <-interrupted:
			break loop
		}
	}

	fmt.Fprintln(os.Stderr, "terminated")
}


```

