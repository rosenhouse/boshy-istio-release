package store

import (
	"log"
	"net"
	"strings"
	"sync"
)

func New() *Repo {
	return &Repo{
		mappings: make(map[string]net.IP),
	}
}

type Repo struct {
	mappings map[string]net.IP
	mu       sync.Mutex
}

func (r *Repo) ReplaceAll(newMappings map[string]net.IP) {
	r.mu.Lock()
	r.mappings = newMappings
	r.mu.Unlock()

	count := len(newMappings)
	log.Printf("store: refreshed %d mappings: %+v", count, newMappings)
}

func (r *Repo) Lookup(hostname string) net.IP {
	hostname = strings.TrimSuffix(hostname, ".") // bosh DNS puts it there, pilot does not
	r.mu.Lock()
	vip, ok := r.mappings[hostname]
	r.mu.Unlock()

	if ok {
		log.Printf("store: lookup %q found %q", hostname, vip)
	} else {
		log.Printf("store: lookup %q found nothing", hostname)
	}
	return vip
}
