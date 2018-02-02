package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"virtual-ip-agent/config"
	"virtual-ip-agent/localdns"
	"virtual-ip-agent/pilot"
)

func main() {
	err := mainWithErr()
	if err != nil {
		log.Fatalf("%s", err)
	}
}

func mainWithErr() error {
	var cfgFilePath string
	flag.StringVar(&cfgFilePath, "config", "", "path to config file")
	flag.Parse()
	cfg, err := config.Load(cfgFilePath)
	if err != nil {
		return fmt.Errorf("loading config: %s", err)
	}

	_, expectedCIDR, err := net.ParseCIDR(cfg.VirtualIPCIDR)
	if err != nil {
		return fmt.Errorf("unable to parse virtual ip cidr: %s", err)
	}

	refreshInterval, err := time.ParseDuration(cfg.RefreshInterval)
	if err != nil {
		return fmt.Errorf("parsing refresh interval: %s", err)
	}

	pilotClient := &pilot.Client{
		ListenersURL:      pilot.GetListenersURL(cfg.PilotBaseURL, cfg.LocalIP),
		ExpectedDNSSuffix: cfg.TLD,
		ExpectedCIDR:      expectedCIDR,
	}

	dnsUpdater := &localdns.Updater{
		OurTLD: cfg.TLD,
	}

	for {
		mappings, err := pilotClient.GetMappings()
		if err != nil {
			return fmt.Errorf("get mappings: %s", err)
		}

		err = dnsUpdater.Sync(mappings)
		if err != nil {
			return fmt.Errorf("local dns sync: %s", err)
		}

		time.Sleep(refreshInterval)
	}
}
