package registerer

import "github.com/google/uuid"

type Registerer interface {
	Register() (uuid.UUID, error)
}
