package model

type Endpoint struct {
	Name string `json:"name,omitempty"`
	Host string `json:"host,omitempty"`
	IPv4 string `json:"ipv4,omitempty"`
	IPv6 string `json:"ipv6,omitempty"`
	Port int    `json:"port,omitempty"`
}
