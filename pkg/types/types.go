package types

import (
	"time"

	"github.com/google/uuid"
)

// BroadcastPacket represents a single broadcast packet sent by the server
type BroadcastPacket struct {
	ID        uuid.UUID `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Data      []byte    `json:"data"`
}

// Client represents a registered broadcast receiver
type Client struct {
	ID        string            `json:"id"`
	IP        string            `json:"ip"`
	LastSeen  time.Time         `json:"last_seen"`
	Connected bool              `json:"connected"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// BroadcastReport represents a client's report of receiving a broadcast
type BroadcastReport struct {
	ClientID  string    `json:"client_id"`
	PacketID  uuid.UUID `json:"packet_id"`
	Timestamp time.Time `json:"timestamp"`
}

// ServerConfig holds the server's configuration
type ServerConfig struct {
	HTTPPort           int           `json:"http_port"`
	HTTPInterface      string        `json:"http_interface"` // Interface name for HTTP traffic
	BroadcastPort      int           `json:"broadcast_port"`
	BroadcastInterface string        `json:"broadcast_interface"` // Interface name for broadcast traffic
	Timeout            time.Duration `json:"timeout"`
}

// ClientConfig holds the client's configuration
type ClientConfig struct {
	ServerURL          string `json:"server_url"` // Full URL including interface IP
	BroadcastPort      int    `json:"broadcast_port"`
	BroadcastInterface string `json:"broadcast_interface"` // Interface name for broadcast traffic
	ClientID           string `json:"client_id"`
}
