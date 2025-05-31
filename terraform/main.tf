terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

# Configure the Digital Ocean Provider
provider "digitalocean" {
  # Token will be read from DIGITALOCEAN_TOKEN environment variable
}

# Create a VPC for our network
resource "digitalocean_vpc" "broadcast_network" {
  name     = "broadcast-test-network"
  region   = "nyc1"  # You can change this to your preferred region
  ip_range = "10.0.0.0/16"
}

# Create the server droplet
resource "digitalocean_droplet" "server" {
  image    = "ubuntu-22-04-x64"
  name     = "broadcast-server"
  region   = digitalocean_vpc.broadcast_network.region
  size     = "s-2vcpu-2gb"  # 2GB RAM, 2 vCPUs
  vpc_uuid = digitalocean_vpc.broadcast_network.id
  ssh_keys = [digitalocean_ssh_key.broadcast_key.fingerprint]

  user_data = <<-EOF
              #!/bin/bash
              apt-get update
              apt-get install -y golang-go git make
              
              # Configure Go environment
              mkdir -p /root/go
              mkdir -p /root/.cache/go-build
              echo 'export GOPATH=/root/go' >> /root/.bashrc
              echo 'export PATH=$PATH:/root/go/bin' >> /root/.bashrc
              echo 'export GOCACHE=/root/.cache/go-build' >> /root/.bashrc
              echo 'export HOME=/root' >> /root/.bashrc
              export GOPATH=/root/go
              export PATH=$PATH:/root/go/bin
              export GOCACHE=/root/.cache/go-build
              export HOME=/root
              
              # Clone and build
              git clone https://github.com/thomasdaly5/broadcast-logger.git /root/broadcast-logger
              cd /root/broadcast-logger
              go mod download
              make build
              
              # Start server with eth1 as broadcast interface
              nohup ./bin/server -port 8080 -broadcast-port 9999 -broadcast-interface eth1 -broadcast-ip 10.0.0.255 > /var/log/broadcast-server.log 2>&1 &
              EOF
}

# Create client droplets
resource "digitalocean_droplet" "clients" {
  count    = 8
  image    = "ubuntu-22-04-x64"
  name     = "broadcast-client-${count.index + 1}"
  region   = digitalocean_vpc.broadcast_network.region
  size     = "s-1vcpu-1gb"  # 1GB RAM, 1 vCPU
  vpc_uuid = digitalocean_vpc.broadcast_network.id
  ssh_keys = [digitalocean_ssh_key.broadcast_key.fingerprint]

  user_data = <<-EOF
              #!/bin/bash
              apt-get update
              apt-get install -y golang-go git make
              
              # Configure Go environment
              mkdir -p /root/go
              mkdir -p /root/.cache/go-build
              echo 'export GOPATH=/root/go' >> /root/.bashrc
              echo 'export PATH=$PATH:/root/go/bin' >> /root/.bashrc
              echo 'export GOCACHE=/root/.cache/go-build' >> /root/.bashrc
              echo 'export HOME=/root' >> /root/.bashrc
              export GOPATH=/root/go
              export PATH=$PATH:/root/go/bin
              export GOCACHE=/root/.cache/go-build
              export HOME=/root
              
              # Clone and build
              git clone https://github.com/thomasdaly5/broadcast-logger.git /root/broadcast-logger
              cd /root/broadcast-logger
              go mod download
              make build
              
              # Start client with eth1 as broadcast interface
              nohup ./bin/client -server http://${digitalocean_droplet.server.ipv4_address}:8080 -id client-${count.index + 1} -broadcast-interface eth1 > /var/log/broadcast-client-${count.index + 1}.log 2>&1 &
              EOF
}

# Create SSH key for droplet access
resource "digitalocean_ssh_key" "broadcast_key" {
  name       = "broadcast-test-key"
  public_key = file("${path.module}/ssh/broadcast.pub")
}

# Output the server's private IP (for client configuration)
output "server_private_ip" {
  value = digitalocean_droplet.server.ipv4_address_private
}

# Output the server's public IP (for external access)
output "server_public_ip" {
  value = digitalocean_droplet.server.ipv4_address
}

# Output all client private IPs
output "client_private_ips" {
  value = digitalocean_droplet.clients[*].ipv4_address_private
} 
