variable "do_region" {
  description = "Digital Ocean region to deploy resources"
  type        = string
  default     = "nyc1"
}

variable "client_count" {
  description = "Number of client instances to create"
  type        = number
  default     = 8
}

variable "server_size" {
  description = "Size of the server droplet"
  type        = string
  default     = "s-2vcpu-2gb"
}

variable "client_size" {
  description = "Size of the client droplets"
  type        = string
  default     = "s-1vcpu-1gb"
}

variable "server_port" {
  description = "Port for the server's HTTP interface"
  type        = number
  default     = 8080
}

variable "broadcast_port" {
  description = "Port for broadcast traffic"
  type        = number
  default     = 9999
} 
