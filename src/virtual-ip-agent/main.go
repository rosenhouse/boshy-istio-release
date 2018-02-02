package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"virtual-ip-agent/config"
	"virtual-ip-agent/handlers"
	"virtual-ip-agent/pilot"
	"virtual-ip-agent/store"
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

	store := store.New()

	boshDNSHandler := &handlers.BoshDNSAdapter{
		Store: store,
	}

	go func() {
		for {
			mappings, err := pilotClient.GetMappings()
			if err != nil {
				log.Fatalf("get mappings: %s", err)
			}

			store.ReplaceAll(mappings)

			time.Sleep(refreshInterval)
		}
	}()

	listenAddr := fmt.Sprintf("127.0.0.1:%d", cfg.ListenPort)
	return http.ListenAndServe(listenAddr, boshDNSHandler)
}
