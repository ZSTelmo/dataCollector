package models

// WriteOptions contains configuration for CSV writing
type WriteOptions struct {
	Directory  string
	Filename   string
	AppendDate bool
}
