package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
)

type Config struct {
	Port         int
	TTL          time.Duration
	AllowedPeers *net.IPNet
}

type element struct {
	EnvVar       string
	DefaultValue string
	Parser       func(*Config, string) error
}

var elements = []element{
	{
		"PORT", "8080", func(c *Config, s string) (e error) {
			c.Port, e = strconv.Atoi(s)
			return
		},
	},
	{
		"TTL", "30s", func(c *Config, s string) (e error) {
			c.TTL, e = time.ParseDuration(s)
			return
		},
	},
	{
		"ALLOWED_PEERS", "0.0.0.0/0", func(c *Config, s string) (e error) {
			_, c.AllowedPeers, e = net.ParseCIDR(s)
			return
		},
	},
}

func GetConfig(logger lager.Logger, environ []string) (*Config, error) {
	config := &Config{}
	envMap := make(map[string]string)
	for _, ev := range environ {
		a := strings.SplitN(ev, "=", 2)
		envMap[a[0]] = a[1]
	}

	for _, el := range elements {
		val := envMap[el.EnvVar]
		if val == "" {
			val = el.DefaultValue
		}

		if err := el.Parser(config, val); err != nil {
			return config, fmt.Errorf("unable to parse %q: %s", el.EnvVar, err)
		}

		logger.Info("parsed-config", lager.Data{el.EnvVar: val})
	}

	return config, nil
}
