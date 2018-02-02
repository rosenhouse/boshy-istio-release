package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"time"

	"gopkg.in/validator.v2"
)

type Config struct {
	RefreshInterval string `json:"refresh_interval" validate:"nonzero"`
	VirtualIPCIDR   string `json:"virtual_ip_cidr" validate:"nonzero"`
	TLD             string `json:"tld" validate:"nonzero"`
	PilotBaseURL    string `json:"pilot_base_url" validate:"nonzero"`
	LocalIP         string `json:"local_ip" validate:"nonzero"`
	ListenPort      int    `json:"listen_port" validate:"nonzero"`
}

func Load(path string) (*Config, error) {
	configBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = json.Unmarshal(configBytes, &cfg)
	if err != nil {
		return nil, fmt.Errorf("parsing config: %s", err)
	}

	err = validator.Validate(cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %s", err)
	}

	// some basic sanity checks
	_, err = time.ParseDuration(cfg.RefreshInterval)
	if err != nil {
		return nil, fmt.Errorf("parsing refresh interval: %s", err)
	}

	_, _, err = net.ParseCIDR(cfg.VirtualIPCIDR)
	if err != nil {
		return nil, fmt.Errorf("parsing virtual_ip_cidr: %s", err)
	}

	if len(cfg.TLD) < 2 {
		return nil, fmt.Errorf("invalid tld %q", cfg.TLD)
	}

	if !strings.HasPrefix(cfg.PilotBaseURL, "http") {
		return nil, fmt.Errorf("invalid pilot_base_url: %s", cfg.PilotBaseURL)
	}

	if net.ParseIP(cfg.LocalIP) == nil {
		return nil, fmt.Errorf("invalid local_ip: %s", cfg.LocalIP)
	}

	if cfg.ListenPort < 1 {
		return nil, fmt.Errorf("invalid listen port %d", cfg.ListenPort)
	}

	log.Printf("loaded config: %+v", cfg)

	return &cfg, nil
}
