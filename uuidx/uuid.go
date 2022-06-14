package uuidx

import "github.com/gofrs/uuid"

// NewV4 returns a new randomly generated UUID or panics.
func NewV4() uuid.UUID {
	return uuid.Must(uuid.NewV4())
}
