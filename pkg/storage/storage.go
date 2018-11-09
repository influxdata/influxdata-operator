package storage

import "io"

type StorageInterface interface {
	// Store creates a new object in the underlying provider's datastore if it does not exist,
	// or replaces the existing object if it does exist.
	Store(key string, body io.ReadCloser) error
	// Retrieve return the object in the underlying provider's datastore if it exists.
	Retrieve(key string) (io.ReadCloser, error)
}