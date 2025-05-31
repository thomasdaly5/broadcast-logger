# Broadcast Logger

A system to help debug Ethernet broadcast packet delivery in a Layer 2 network.

## Overview

This application consists of two components:

1. **Broadcast Sender (Server)**
   - Sends broadcast packets with unique identifiers
   - Runs an HTTP server for client registration
   - Tracks connected clients
   - Monitors broadcast packet delivery
   - Reports delivery statistics

2. **Broadcast Receiver (Client)**
   - Registers with the server via HTTP
   - Listens for broadcast packets
   - Reports received packets back to the server

## Building

To build both the server and client:

```bash
make build
```

This will create two binaries in the `bin` directory:
- `bin/server` - The broadcast sender
- `bin/client` - The broadcast receiver

## Running

### 1. Start the Server

```bash
./bin/server
```

By default, the server:
- Listens on HTTP port 8080
- Sends broadcasts on UDP port 9999
- Waits 5 seconds for client reports

You can customize these settings:
```bash
./bin/server -port 8080 -broadcast-port 9999 -timeout 5s
```

### 2. Start One or More Clients

```bash
./bin/client -server http://localhost:8080
```

By default, the client:
- Connects to the server at http://localhost:8080
- Listens for broadcasts on UDP port 9999
- Generates a random client ID

You can customize these settings:
```bash
./bin/client -server http://server-ip:8080 -broadcast-port 9999 -id custom-client-id
```

### 3. Testing Broadcast Delivery

To send a broadcast packet, make a POST request to the server's broadcast endpoint:

```bash
curl -X POST http://localhost:8080/broadcast
```

The server will:
1. Send a broadcast packet
2. Wait for client reports
3. Log which clients received the packet

You can check the current status of all clients and the last broadcast at:

```bash
curl http://localhost:8080/status
```

## Troubleshooting

1. **Firewall Issues**
   - Ensure UDP port 9999 is open for both sending and receiving
   - Ensure HTTP port 8080 is open for client-server communication

2. **Network Configuration**
   - Verify that broadcast packets are allowed on your network
   - Check that clients are on the same Layer 2 network segment

3. **Client Connection**
   - Verify the server URL is correct when starting clients
   - Check that clients can reach the server's HTTP endpoint

## Development

To clean build artifacts:
```bash
make clean
```

To run tests:
```bash
make test
```

