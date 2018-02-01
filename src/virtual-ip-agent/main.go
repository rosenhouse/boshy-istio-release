package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
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
	cfg, err := LoadConfig(cfgFilePath)
	if err != nil {
		return fmt.Errorf("loading config: %s", err)
	}

	_, expectedCIDR, err := net.ParseCIDR(cfg.VirtualIPCIDR)
	if err != nil {
		return fmt.Errorf("unable to parse virtual ip cidr: %s", err)
	}

	converger := &Converger{
		listenersURL:      getListenersURL(cfg.PilotBaseURL, cfg.LocalIP),
		expectedDNSSuffix: cfg.TLD,
		expectedCIDR:      expectedCIDR,
	}

	for {
		err := converger.convergeOnce()
		if err != nil {
			return fmt.Errorf("converge failure: %s", err)
		}
	}
}

func getListenersURL(pilotBaseURL, localIP string) string {
	const template = "%s/v1/listeners/x/sidecar~%s~x~x"
	return fmt.Sprintf(template, pilotBaseURL, localIP)
}

type Converger struct {
	listenersURL      string
	expectedDNSSuffix string
	expectedCIDR      *net.IPNet
}

func (c *Converger) convergeOnce() error {
	resp, err := http.Get(c.listenersURL)
	if err != nil {
		return fmt.Errorf("get listeners: %s", err)
	}
	defer resp.Body.Close()

	var respStruct ListenersResponse
	err = json.NewDecoder(resp.Body).Decode(&respStruct)
	if err != nil {
		return fmt.Errorf("parsing response: %s", err)
	}

	for _, listener := range respStruct.Listeners {
		hostname, vip, err := c.tryInferMapping(listener)
		if err != nil {
			return fmt.Errorf("try infer mapping: %s", err)
		}
		if hostname == "" {
			continue
		}
		fmt.Printf("%s\t%s\n", vip, hostname)
	}

	return nil
}

type Hostname string

// a terrible hack
func (c *Converger) tryInferMapping(listener TCPListener) (Hostname, net.IP, error) {
	name := listener.Name
	if len(listener.Filters) == 0 {
		log.Printf("%s: no filters", name)
		return "", nil, nil
	}
	filter := listener.Filters[0]
	if len(filter.Config.RouteConfig.Routes) == 0 {
		log.Printf("%s: no routes", name)
		return "", nil, nil
	}
	route := filter.Config.RouteConfig.Routes[0]
	if !strings.HasPrefix(route.Cluster, "out.") {
		log.Printf("%s: wrong cluster prefix", name)
		return "", nil, nil
	}
	if len(route.DestinationIPList) == 0 {
		log.Printf("%s: no destination IPs", name)
		return "", nil, nil
	}
	dstIPString := route.DestinationIPList[0]
	if !strings.HasSuffix(dstIPString, "/32") {
		log.Printf("%s: wrong ip subnet mask size", name)
		return "", nil, nil
	}
	vip, _, err := net.ParseCIDR(dstIPString)
	if err != nil {
		log.Printf("%s: unable to parse dest ip as cidr", name)
		return "", nil, nil
	}
	if !c.expectedCIDR.Contains(vip) {
		log.Printf("%s: vip not contained in expected cidr range", name)
		return "", nil, nil
	}

	// e.g. "out.example-httpbin.banana.sample-deployment.boshy|other-http",
	clusterNameParts := strings.Split(route.Cluster, "|")
	if len(clusterNameParts) != 2 {
		log.Printf("%s: cluster name format", name)
		return "", nil, nil
	}
	hostname := strings.TrimPrefix(clusterNameParts[0], "out.")
	if !strings.HasSuffix(hostname, c.expectedDNSSuffix) {
		return "", nil, fmt.Errorf("failed parsing DNS name from %s", listener.Name)
	}
	if vip == nil {
		return "", nil, fmt.Errorf("failed parsing vip from %s", listener.Name)
	}
	return Hostname(hostname), vip, nil
}

type ListenersResponse struct {
	Listeners []TCPListener
}

type TCPListener struct {
	Name    string
	Address string
	Filters []TCPFilter
}

type TCPFilter struct {
	Config TCPProxyFilterConfig
}

type TCPProxyFilterConfig struct {
	RouteConfig TCPRouteConfig `json:"route_config"`
}

type TCPRouteConfig struct {
	Routes []TCPRoute `json:"routes"`
}

type TCPRoute struct {
	Cluster           string   `json:"cluster"`
	DestinationIPList []string `json:"destination_ip_list,omitempty"`
	DestinationPorts  string   `json:"destination_ports,omitempty"`
	SourceIPList      []string `json:"source_ip_list,omitempty"`
	SourcePorts       string   `json:"source_ports,omitempty"`
}
