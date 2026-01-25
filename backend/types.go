package main

import (
	"sort"
	"time"
)

type Host struct {
	Name     string     `yaml:"name" json:"name"`
	Endpoint string     `yaml:"endpoint" json:"endpoint"`
	TLS      *TLSConfig `yaml:"tls,omitempty" json:"tls,omitempty"`
}

type TLSConfig struct {
	CA   string `yaml:"ca" json:"ca"`
	Cert string `yaml:"cert" json:"cert"`
	Key  string `yaml:"key" json:"key"`
}

type Defaults struct {
	BaseDomain string `yaml:"base_domain" json:"base_domain"`
	Scheme     string `yaml:"scheme" json:"scheme"`
}

type ProxyTarget struct {
	Host      string
	Container string
	Port      int
}

type Config struct {
	Hosts          []Host                         `yaml:"hosts" json:"hosts"`
	ProxyMappings  map[string]string              `yaml:"proxy_mappings" json:"proxy_mappings"`
	ProxyRoutes    map[string]map[string][]string `json:"proxy_routes"`
	HostAddresses  map[string]string              `yaml:"host_addresses" json:"host_addresses"`
	Defaults       Defaults                       `yaml:"defaults" json:"defaults"`
	ParsedMappings map[string]ProxyTarget         `json:"-"`
}

type Port struct {
	Private uint16 `json:"private"`
	Public  uint16 `json:"public"`
	Type    string `json:"type"`
	Proxied bool   `json:"proxied"`
}

type Container struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	State     string            `json:"state"`
	Status    string            `json:"status"`
	Host      string            `json:"host"`
	Ports     []Port            `json:"ports"`
	Labels    map[string]string `json:"labels"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// SortPorts sorts ports by private port number (ascending) and ensures consistent ordering
func (c *Container) SortPorts() {
	sort.Slice(c.Ports, func(i, j int) bool {
		if c.Ports[i].Private != c.Ports[j].Private {
			return c.Ports[i].Private < c.Ports[j].Private
		}
		return c.Ports[i].Public < c.Ports[j].Public
	})
}
