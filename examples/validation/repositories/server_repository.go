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
	Hostname   string `json:"hostname" validate:"required,alpha"`
	IpAddress  string `json:"ip_address" validate:"required,ip"`
	MacAddress string `json:"mac_address" validate:"required,mac"`
	Port       string `json:"port" validate:"required,numeric"`
	Version    string `json:"version" validate:"omitempty,semver"`
}

// ServerRepository defines the contract for server persistence operations.
type ServerRepository interface {
	Create(ctx context.Context, entity *Server) error
	Get(ctx context.Context, id uint) (*Server, error)
	List(ctx context.Context, params ListParams) (*ListResult[*Server], error)
	Update(ctx context.Context, entity *Server) error
	Delete(ctx context.Context, id uint) error
}
