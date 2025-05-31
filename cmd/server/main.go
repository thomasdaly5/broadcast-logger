package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/tjd/broadcast-logger/pkg/types"
)

var (
	config = types.ServerConfig{
		HTTPPort:      8080,
		BroadcastPort: 9999,
		Timeout:       5 * time.Second,
	}

	clients     = make(map[string]*types.Client)
	clientsLock sync.RWMutex

	currentBroadcast *types.BroadcastPacket
	broadcastLock    sync.RWMutex
)

// Add these types at the top with other vars
type broadcastResult struct {
	PacketID     uuid.UUID         `json:"packet_id"`
	SentAt       time.Time         `json:"sent_at"`
	ReceivedBy   map[string]string `json:"received_by"` // client ID -> received timestamp
	MissedBy     []string          `json:"missed_by"`   // list of client IDs that missed it
	TotalClients int               `json:"total_clients"`
	Timeout      time.Duration     `json:"timeout"`
}

// Add this type near the top with other types
type clientStatus struct {
	types.Client
	LastPacketID string `json:"last_packet_id,omitempty"`
}

// Add these helper functions at the top level
func getInterfaceIP(ifaceName string) (net.IP, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("interface %s not found: %v", ifaceName, err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("failed to get addresses for interface %s: %v", ifaceName, err)
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP, nil
		}
	}
	return nil, fmt.Errorf("no IPv4 address found on interface %s", ifaceName)
}

func main() {
	// Parse command line flags
	flag.IntVar(&config.HTTPPort, "port", config.HTTPPort, "HTTP server port")
	flag.StringVar(&config.HTTPInterface, "http-iface", "", "Network interface for HTTP traffic")
	flag.IntVar(&config.BroadcastPort, "broadcast-port", config.BroadcastPort, "Broadcast port")
	flag.StringVar(&config.BroadcastInterface, "broadcast-iface", "", "Network interface for broadcast traffic")
	flag.DurationVar(&config.Timeout, "timeout", config.Timeout, "Broadcast timeout")
	flag.Parse()

	// Get HTTP interface IP
	var httpIP net.IP
	var err error
	if config.HTTPInterface != "" {
		httpIP, err = getInterfaceIP(config.HTTPInterface)
		if err != nil {
			log.Fatalf("Failed to get HTTP interface IP: %v", err)
		}
		log.Printf("Using HTTP interface %s with IP %s", config.HTTPInterface, httpIP)
	}

	// Set up router
	r := mux.NewRouter()

	// API endpoints
	r.HandleFunc("/register", handleRegister).Methods("POST")
	r.HandleFunc("/report", handleReport).Methods("POST")
	r.HandleFunc("/status", handleStatus).Methods("GET")
	r.HandleFunc("/broadcast", handleBroadcast).Methods("POST")

	// Start HTTP server on specific interface if specified
	addr := fmt.Sprintf("%s:%d", httpIP, config.HTTPPort)
	if httpIP == nil {
		addr = fmt.Sprintf(":%d", config.HTTPPort)
	}
	log.Printf("Starting HTTP server on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	var client types.Client
	if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	clientsLock.Lock()
	defer clientsLock.Unlock()

	client.LastSeen = time.Now()
	client.Connected = true
	clients[client.ID] = &client

	log.Printf("Client registered: %s from %s", client.ID, client.IP)
	w.WriteHeader(http.StatusOK)
}

func handleReport(w http.ResponseWriter, r *http.Request) {
	var report types.BroadcastReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	clientsLock.Lock()
	defer clientsLock.Unlock()

	if client, exists := clients[report.ClientID]; exists {
		client.LastSeen = report.Timestamp
		// Store the packet ID in the client's metadata
		if client.Metadata == nil {
			client.Metadata = make(map[string]string)
		}
		client.Metadata["last_packet_id"] = report.PacketID.String()
		log.Printf("Received report from %s for packet %s at %s",
			report.ClientID, report.PacketID, report.Timestamp.Format(time.RFC3339))
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Error(w, "Client not registered", http.StatusNotFound)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	clientsLock.RLock()
	defer clientsLock.RUnlock()

	// Create a map of client statuses
	clientStatuses := make(map[string]clientStatus)
	for id, client := range clients {
		cs := clientStatus{
			Client: *client,
		}
		if client.Metadata != nil {
			if lastPacketID, exists := client.Metadata["last_packet_id"]; exists {
				cs.LastPacketID = lastPacketID
			}
		}
		clientStatuses[id] = cs
	}

	status := struct {
		Clients          map[string]clientStatus `json:"clients"`
		CurrentBroadcast *types.BroadcastPacket  `json:"current_broadcast,omitempty"`
		LastPacketID     string                  `json:"last_packet_id,omitempty"`
	}{
		Clients:          clientStatuses,
		CurrentBroadcast: currentBroadcast,
		LastPacketID: func() string {
			if currentBroadcast != nil {
				return currentBroadcast.ID.String()
			}
			return ""
		}(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func handleBroadcast(w http.ResponseWriter, r *http.Request) {
	// Get broadcast interface IP
	var broadcastIP net.IP
	var err error
	if config.BroadcastInterface != "" {
		broadcastIP, err = getInterfaceIP(config.BroadcastInterface)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get broadcast interface IP: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("Using broadcast interface %s with IP %s", config.BroadcastInterface, broadcastIP)
	}

	// Create a new broadcast packet
	packet := &types.BroadcastPacket{
		ID:        uuid.New(),
		Timestamp: time.Now(),
		Data:      []byte("broadcast test"),
	}

	// Create a result tracker
	result := &broadcastResult{
		PacketID:   packet.ID,
		SentAt:     packet.Timestamp,
		ReceivedBy: make(map[string]string),
		MissedBy:   make([]string, 0),
		Timeout:    config.Timeout,
	}

	// Get list of connected clients before sending
	clientsLock.RLock()
	connectedClients := make(map[string]bool)
	for id, client := range clients {
		if client.Connected {
			connectedClients[id] = true
			result.MissedBy = append(result.MissedBy, id)
		}
	}
	result.TotalClients = len(connectedClients)
	clientsLock.RUnlock()

	// Serialize and send the packet
	data, err := json.Marshal(packet)
	if err != nil {
		http.Error(w, "Failed to marshal packet", http.StatusInternalServerError)
		return
	}

	// Update the broadcast address to use the specific interface
	var broadcastAddr string
	if broadcastIP != nil {
		broadcastAddr = fmt.Sprintf("%s:%d", broadcastIP, config.BroadcastPort)
	} else {
		broadcastAddr = fmt.Sprintf("255.255.255.255:%d", config.BroadcastPort)
	}

	// Create a UDP connection on the specific interface
	var conn net.Conn
	if broadcastIP != nil {
		// Create a UDP connection on the specific interface
		addr := &net.UDPAddr{
			IP:   broadcastIP,
			Port: 0, // Let the system choose a port
		}
		udpConn, err := net.ListenUDP("udp4", addr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create UDP connection: %v", err), http.StatusInternalServerError)
			return
		}
		defer udpConn.Close()

		// Enable broadcast on the socket
		file, err := udpConn.File()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get socket file: %v", err), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		if err := syscall.SetsockoptInt(int(file.Fd()), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1); err != nil {
			http.Error(w, fmt.Sprintf("Failed to set broadcast flag: %v", err), http.StatusInternalServerError)
			return
		}
		conn = udpConn
	} else {
		// Fall back to default broadcast behavior
		var err error
		conn, err = net.Dial("udp4", broadcastAddr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to dial UDP: %v", err), http.StatusInternalServerError)
			return
		}
		defer conn.Close()
	}

	// Send the broadcast
	_, err = conn.Write(data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to send broadcast: %v", err), http.StatusInternalServerError)
		return
	}

	// Store the current broadcast
	broadcastLock.Lock()
	currentBroadcast = packet
	broadcastLock.Unlock()

	log.Printf("Broadcast sent: %s to %d clients", packet.ID, result.TotalClients)

	// Wait for reports with timeout
	timeout := time.After(config.Timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// Timeout reached, prepare final result
			clientsLock.RLock()
			for id := range connectedClients {
				if _, received := result.ReceivedBy[id]; !received {
					result.MissedBy = append(result.MissedBy, id)
				}
			}
			clientsLock.RUnlock()

			// Remove duplicates from MissedBy
			seen := make(map[string]bool)
			uniqueMissed := make([]string, 0)
			for _, id := range result.MissedBy {
				if !seen[id] {
					seen[id] = true
					uniqueMissed = append(uniqueMissed, id)
				}
			}
			result.MissedBy = uniqueMissed

			// Return the result
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
			log.Printf("Broadcast %s complete. Received by %d/%d clients",
				packet.ID, len(result.ReceivedBy), result.TotalClients)
			return

		case <-ticker.C:
			// Check for new reports
			clientsLock.RLock()
			for id, client := range clients {
				if client.Connected && client.LastSeen.After(packet.Timestamp) {
					if _, exists := result.ReceivedBy[id]; !exists {
						result.ReceivedBy[id] = client.LastSeen.Format(time.RFC3339)
						// Remove from missed list
						for i, missedID := range result.MissedBy {
							if missedID == id {
								result.MissedBy = append(result.MissedBy[:i], result.MissedBy[i+1:]...)
								break
							}
						}
					}
				}
			}
			clientsLock.RUnlock()
		}
	}
}
