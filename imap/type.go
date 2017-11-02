package imap

// MailboxInfo contains a mailbox's basic information.
type MailboxInfo struct {
	Name   string
	Exists int
	Recent int
	Flags  []string
}

// FetchResult contains the response of FETCH command.
type FetchResult struct {
	InternalDate string
	Flags        []string
	Data         []byte
}
