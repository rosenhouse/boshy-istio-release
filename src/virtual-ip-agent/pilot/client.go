package pilot

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

func GetListenersURL(pilotBaseURL, localIP string) string {
	const template = "%s/v1/listeners/x/sidecar~%s~x~x"
	return fmt.Sprintf(template, pilotBaseURL, localIP)
}

type Client struct {
	ListenersURL      string
	ExpectedDNSSuffix string
	ExpectedCIDR      *net.IPNet
}

// Queries Pilot LDS and returns a map of virtual Hostname to virtual IP
func (c *Client) GetMappings() (map[string]net.IP, error) {
	resp, err := http.Get(c.ListenersURL)
	if err != nil {
		return nil, fmt.Errorf("get listeners: %s", err)
	}
	defer resp.Body.Close()

	var respStruct ListenersResponse
	err = json.NewDecoder(resp.Body).Decode(&respStruct)
	if err != nil {
		return nil, fmt.Errorf("parsing response: %s", err)
	}

	ret := map[string]net.IP{}
	for _, listener := range respStruct.Listeners {
		hostname, vip, err := c.tryInferMapping(listener)
		if err != nil {
			return nil, fmt.Errorf("try infer mapping: %s", err)
		}
		if hostname == "" {
			continue
		}
		ret[hostname] = vip
	}

	return ret, nil
}

// a sequence of terrible hacks
func (c *Client) tryInferMapping(listener TCPListener) (string, net.IP, error) {
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
	if !c.ExpectedCIDR.Contains(vip) {
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
	if !strings.HasSuffix(hostname, c.ExpectedDNSSuffix) {
		return "", nil, fmt.Errorf("failed parsing DNS name from %s", listener.Name)
	}
	if vip == nil {
		return "", nil, fmt.Errorf("failed parsing vip from %s", listener.Name)
	}
	return hostname, vip, nil
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
