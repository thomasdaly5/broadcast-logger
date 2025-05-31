# Broadcast Logger - Terraform Infrastructure

This directory contains Terraform configuration to deploy the Broadcast Logger system on Digital Ocean.

## Prerequisites

1. Install Terraform (https://www.terraform.io/downloads.html)
2. Create a Digital Ocean account
3. Generate a Digital Ocean API token (https://cloud.digitalocean.com/account/api/tokens)
4. Generate an SSH key pair for droplet access

## Setup

1. Generate an SSH key pair:
   ```bash
   mkdir -p ssh
   ssh-keygen -t rsa -b 4096 -f ssh/broadcast -N ""
   ```

2. Set your Digital Ocean API token:
   ```bash
   export DIGITALOCEAN_TOKEN="your-api-token-here"
   ```

3. Initialize Terraform:
   ```bash
   terraform init
   ```

4. Review the planned changes:
   ```bash
   terraform plan
   ```

5. Apply the configuration:
   ```bash
   terraform apply
   ```

## Infrastructure Details

The configuration creates:
- 1x Server droplet (2GB RAM, 2 vCPUs)
- 128x Client droplets (1GB RAM, 1 vCPU each)
- A private VPC network for all droplets
- SSH key for secure access

## Customization

You can customize the deployment by modifying the variables in `variables.tf` or by passing them during apply:

```bash
terraform apply -var="client_count=64" -var="do_region=sfo3"
```

Available variables:
- `do_region`: Digital Ocean region (default: nyc1)
- `client_count`: Number of client instances (default: 128)
- `server_size`: Server droplet size (default: s-2vcpu-2gb)
- `client_size`: Client droplet size (default: s-1vcpu-1gb)
- `server_port`: HTTP server port (default: 8080)
- `broadcast_port`: Broadcast traffic port (default: 9999)

## Accessing the Infrastructure

After applying the configuration, Terraform will output:
- Server's public IP (for external access)
- Server's private IP (for client communication)
- List of client private IPs

To access the server:
```bash
ssh -i ssh/broadcast root@<server_public_ip>
```

## Cleanup

To destroy all created resources:
```bash
terraform destroy
```

## Cost Estimation

Current configuration (as of 2024):
- Server (s-2vcpu-2gb): $12/month
- Clients (s-1vcpu-1gb): $6/month each
- Total for 128 clients: $780/month

Consider using `terraform destroy` when not actively testing to avoid unnecessary costs. 