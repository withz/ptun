package config

import (
	"errors"
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

func InitFile(p string, v any) (err error) {
	viper.SetConfigName(strings.TrimSuffix(p, ".toml"))
	err = viper.ReadInConfig()
	if err != nil {
		return err
	}
	if reflect.TypeOf(v).Kind() != reflect.Ptr {
		return errInvalidConfigVariable
	}
	err = viper.Unmarshal(&com)
	if err != nil {
		return err
	}
	err = viper.Unmarshal(v)
	return err
}

type common struct {
	Name  string `toml:"Name"`
	Token string `toml:"Token"`
}

var (
	com common

	configPaths = []string{
		".",
		"..",
		"./conf",
		"../conf",
		"/etc/ptun",
		"$HOME/.ptun",
		"../..",
		"../../conf",
	}
	errInvalidConfigVariable = errors.New("invalid config variable, need pointer")
	errInvalidStunServerType = errors.New("invalid stun server type")
)

func init() {
	viper.SetConfigType("toml")
	viper.SetEnvPrefix("PTUN")
	for _, p := range configPaths {
		viper.AddConfigPath(p)
	}
}

type StunServerType string

const (
	Simple   StunServerType = "simple"
	Standard StunServerType = "standard"
)

func validateStunServerType(t StunServerType) (err error) {
	if t == Simple {
		return nil
	}
	if t == Standard {
		return nil
	}
	return errInvalidStunServerType
}
