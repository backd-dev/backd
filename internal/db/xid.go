package db

import (
	"github.com/rs/xid"
)

// NewXID generates a new unique XID
// This is the only file that should import github.com/rs/xid
// All other code should use this function
func NewXID() string {
	return xid.New().String()
}
