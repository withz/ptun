package config

func InitClient() (err error) {
	return InitClientPath("ptun-node1.toml")
}

func InitClientPath(f string) (err error) {
	if err = InitFile(f, &c); err != nil {
		return err
	}
	c.common = &com
	return checkClientConfig()
}

func Client() *client {
	return &c
}

type client struct {
	*common

	ServerHost string
	ServerPort int

	Stun struct {
		Type          StunServerType
		Host          string
		PrimaryPort   int
		SecondaryPort int
	} `toml:"Stun"`

	Net struct {
		Tun       string
		IP        string
		AllowNets []string
		Routers   []struct {
			Next     string
			Networks []string
		} `toml:"Routers"`
	} `toml:"Net"`
}

var c client

func checkClientConfig() (err error) {
	return nil
}
