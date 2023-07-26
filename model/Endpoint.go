package model

type Endpoint struct {
	Name string `json:"name"`
	Host string `json:"host"`
	IPv4 string `json:"ipv4"`
	IPv6 string `json:"ipv6"`
	Port int    `json:"port"`
}
