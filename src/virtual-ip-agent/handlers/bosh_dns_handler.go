// handlers provides HTTP handler to support DNS queries over HTTP
// this is queried by BOSH DNS to lookup ips for hostnames
//
// shamelessly ripped off from
// https://github.com/cloudfoundry/cf-app-sd-release/blob/bfd9827acfe64ace8c8c0/src/bosh-dns-adapter/main.go
package handlers

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"golang.org/x/net/dns/dnsmessage"
)

type repo interface {
	Lookup(hostname string) net.IP
}

type BoshDNSAdapter struct {
	Store repo
}

func (h *BoshDNSAdapter) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	dnsType := getQueryParam(req, "type", "1")
	name := getQueryParam(req, "name", "")

	if dnsType != "1" {
		writeResponse(resp, dnsmessage.RCodeSuccess, name, dnsType, nil)
		return
	}

	if name == "" {
		resp.WriteHeader(http.StatusBadRequest)
		writeResponse(resp, dnsmessage.RCodeServerFailure, name, dnsType, nil)
		return
	}

	vip := h.Store.Lookup(name)

	writeResponse(resp, dnsmessage.RCodeSuccess, name, dnsType, []string{vip.String()})
}

func getQueryParam(req *http.Request, key, defaultValue string) string {
	queryValue := req.URL.Query().Get(key)
	if queryValue == "" {
		return defaultValue
	}

	return queryValue
}

func writeResponse(resp http.ResponseWriter, dnsResponseStatus dnsmessage.RCode, requestedInfraName string, dnsType string, ips []string) {
	responseBody, err := buildResponseBody(dnsResponseStatus, requestedInfraName, dnsType, ips)
	if err != nil {
		return
	}

	resp.Write([]byte(responseBody))
}

type Answer struct {
	Name   string `json:"name"`
	RRType uint16 `json:"type"`
	TTL    uint32 `json:"TTL"`
	Data   string `json:"data"`
}

func buildResponseBody(dnsResponseStatus dnsmessage.RCode, requestedInfraName string, dnsType string, ips []string) (string, error) {
	answers := make([]Answer, len(ips), len(ips))
	for i, ip := range ips {
		answers[i] = Answer{
			Name:   requestedInfraName,
			RRType: uint16(dnsmessage.TypeA),
			Data:   ip,
			TTL:    0,
		}
	}

	bytes, err := json.Marshal(answers)
	if err != nil {
		return "", err // not tested
	}

	template := `{
		"Status": %d,
		"TC": false,
		"RD": false,
		"RA": false,
		"AD": false,
		"CD": false,
		"Question":
		[
			{
				"name": "%s",
				"type": %s
			}
		],
		"Answer": %s,
		"Additional": [ ],
		"edns_client_subnet": "0.0.0.0/0"
	}`

	return fmt.Sprintf(template, dnsResponseStatus, requestedInfraName, dnsType, string(bytes)), nil
}
