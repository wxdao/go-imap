package imap

// MailboxInfo contains a mailbox's basic information.
type MailboxInfo struct {
	Name   string
	Exists int
	Recent int
	Flags  []string
}
