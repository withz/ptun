package tools

import (
	"net"

	"github.com/gofrs/uuid"
)

func GenUUID() string {
	u, _ := uuid.NewV4()
	return uuid.Must(u, nil).String()
}

func IsNetError(err error) bool {
	_, ok := err.(net.Error)
	return ok
}
