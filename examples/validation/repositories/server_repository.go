package repositories

import (
	"context"
)

// Server represents a server entity showcasing networking validator types.
//
// Equivalent CLI definition:
//
//	pola generate repository Server hostname:alpha ip_address:ip mac_address:mac port:port version:semver?
type Server struct {
	ID         uint   `json:"id"`
	Hostname   string `json:"hostname" valid:"required,alpha"`
	IpAddress  string `json:"ip_address" valid:"required,ip"`
	MacAddress string `json:"mac_address" valid:"required,mac"`
	Port       string `json:"port" valid:"required,port"`
	Version    string `json:"version" valid:"optional,semver"`
}

// ServerRepository defines the contract for server persistence operations.
type ServerRepository interface {
	Create(ctx context.Context, entity *Server) error
	Get(ctx context.Context, id uint) (*Server, error)
	List(ctx context.Context, params ListParams) (*ListResult[*Server], error)
	Update(ctx context.Context, entity *Server) error
	Delete(ctx context.Context, id uint) error
}
