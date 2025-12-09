package id

import (
	"github.com/google/uuid"
)

func Generate() string {
	return uuid.New().String()
}

func IsValidUUID(id string) (uuid.UUID, error) {
	return uuid.Parse(id)

}
