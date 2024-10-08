package config

import (
	"errors"
)

func InitServer() (err error) {
	return InitServerPath("ptun-hub.toml")
}

func InitServerPath(f string) (err error) {
	if err = InitFile(f, &s); err != nil {
		return err
	}
	s.common = &com
	return checkServerConfig()
}

func Server() *server {
	return &s
}

type server struct {
	*common

	ServerPort int `toml:"ServerPort"`
	Stun       struct {
		Type          StunServerType
		PrimaryPort   int
		SecondaryPort int
	} `toml:"Stun"`
}

var s server

var (
	errCannotUseSameStunPorts = errors.New("cannot use same stun ports")
)

func checkServerConfig() (err error) {
	if err = validateStunServerType(s.Stun.Type); err != nil {
		return err
	}
	if s.Stun.PrimaryPort == s.Stun.SecondaryPort {
		return errCannotUseSameStunPorts
	}
	return nil
}
