package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/tjd/broadcast-logger/pkg/types"
)

var config types.ClientConfig

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
	flag.StringVar(&config.ServerURL, "server", "http://localhost:8080", "Server URL")
	flag.IntVar(&config.BroadcastPort, "broadcast-port", 9999, "Broadcast port")
	flag.StringVar(&config.BroadcastInterface, "broadcast-iface", "", "Network interface for broadcast traffic")
	flag.StringVar(&config.ClientID, "id", uuid.New().String(), "Client ID")
	flag.Parse()

	// Register with the server
	registerWithServer()

	// Listen for broadcast packets
	listenForBroadcasts()
}

func registerWithServer() {
	client := types.Client{
		ID: config.ClientID,
		IP: getLocalIP(),
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(client); err != nil {
		log.Fatalf("Failed to encode client: %v", err)
	}
	resp, err := http.Post(fmt.Sprintf("%s/register", config.ServerURL), "application/json", buf)
	if err != nil {
		log.Fatalf("Failed to register with server: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Server registration failed: %s", resp.Status)
	}
	log.Printf("Registered with server as %s", config.ClientID)
}

func listenForBroadcasts() {
	var broadcastIP net.IP
	var err error
	if config.BroadcastInterface != "" {
		broadcastIP, err = getInterfaceIP(config.BroadcastInterface)
		if err != nil {
			log.Fatalf("Failed to get broadcast interface IP: %v", err)
		}
		log.Printf("Using broadcast interface %s with IP %s", config.BroadcastInterface, broadcastIP)
	}

	addr := net.UDPAddr{
		Port: config.BroadcastPort,
		IP:   broadcastIP,
	}
	if broadcastIP == nil {
		addr.IP = net.IPv4zero
	}

	conn, err := net.ListenUDP("udp4", &addr)
	if err != nil {
		log.Fatalf("Failed to listen for broadcasts: %v", err)
	}
	defer conn.Close()

	if broadcastIP != nil {
		file, err := conn.File()
		if err != nil {
			log.Fatalf("Failed to get socket file: %v", err)
		}
		defer file.Close()

		if err := syscall.SetsockoptInt(int(file.Fd()), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1); err != nil {
			log.Fatalf("Failed to set broadcast flag: %v", err)
		}
	}

	log.Printf("Listening for broadcasts on %s:%d", addr.IP, config.BroadcastPort)

	buf := make([]byte, 2048)
	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error reading UDP: %v", err)
			continue
		}
		log.Printf("Received broadcast from %s: %x", remote, buf[:n])

		// Parse the broadcast packet
		var packet types.BroadcastPacket
		if err := json.Unmarshal(buf[:n], &packet); err != nil {
			log.Printf("Failed to parse broadcast packet: %v", err)
			continue
		}
		log.Printf("Received broadcast packet ID: %s", packet.ID)

		// Report to server
		report := types.BroadcastReport{
			ClientID:  config.ClientID,
			PacketID:  packet.ID,
			Timestamp: time.Now(),
		}
		go reportToServer(report)
	}
}

func reportToServer(report types.BroadcastReport) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(report); err != nil {
		log.Printf("Failed to encode report: %v", err)
		return
	}
	resp, err := http.Post(fmt.Sprintf("%s/report", config.ServerURL), "application/json", buf)
	if err != nil {
		log.Printf("Failed to report to server: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Server report failed: %s", resp.Status)
		return
	}
	log.Printf("Reported packet %s to server", report.PacketID)
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return ""
}
