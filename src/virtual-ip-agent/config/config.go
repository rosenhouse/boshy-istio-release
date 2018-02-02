package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"time"
)

type Config struct {
	RefreshInterval string `json:"refresh_interval"`
	VirtualIPCIDR   string `json:"virtual_ip_cidr"`
	TLD             string `json:"tld"`
	PilotBaseURL    string `json:"pilot_base_url"`
	LocalIP         string `json:"local_ip"`
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

	return &cfg, nil
}
