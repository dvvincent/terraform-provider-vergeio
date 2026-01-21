---
page_title: "VergeIO Experimental Provider"
subcategory: ""
description: |-
  Experimental Terraform provider for VergeOS with advanced features including token auth, multi-tenant management, and parallel file uploads.
---

# VergeIO Experimental Provider (vergeioxp)

The `vergeioxp` provider is an experimental Terraform provider for VergeOS with advanced automation features including:

- **Token-based authentication** — Secure API token auth for CI/CD pipelines
- **Multi-tenant management** — Create and configure tenants, nodes, and storage
- **Parallel file uploads** — Fast image uploads with configurable thread count
- **Auto-networking** — Automatic vnet creation with DHCP and gateway routing
- **Cloud-Init integration** — Built-in credential injection for VMs

## Authentication

The provider supports two authentication methods:

### Option 1: Username/Password
```hcl
provider "vergeioxp" {
  host     = "192.168.1.111"
  username = "admin"
  password = var.verge_password
  insecure = true  # For self-signed certs
}
```

### Option 2: API Token (Recommended for CI/CD)
```hcl
provider "vergeioxp" {
  host     = "192.168.1.111"
  token    = var.verge_token
  insecure = true
}
```

**Creating an API Token:**
```bash
curl -sk -X POST "https://your-vergeos-host/api/sys/tokens" \
  -H "Content-Type: application/json" \
  -d '{"login": "admin", "password": "YourPassword"}'
# Returns: {"location":"/sys/tokens/YOUR_TOKEN_HERE"}
```

## Multi-Tenant Automation Example

This example demonstrates "Automation Inception" — where Terraform provisions a tenant and then configures resources *inside* that tenant:

```hcl
terraform {
  required_providers {
    vergeioxp = {
      source = "dvvincent/vergeioxp"
    }
  }
}

# --- PHASE 1: Host-level resources ---
provider "vergeioxp" {
  alias    = "host"
  host     = "192.168.1.111"
  username = "admin"
  password = var.host_password
  insecure = true
}

# Allocate a public IP for the tenant
resource "vergeioxp_vnet_address" "tenant_ip" {
  provider = vergeioxp.host
  vnet     = 3  # External network
  type     = "virtual"
}

# Create the tenant with auto-routing
resource "vergeioxp_tenant" "corp" {
  provider   = vergeioxp.host
  name       = "corp-tenant"
  password   = var.tenant_admin_password
  ui_address = vergeioxp_vnet_address.tenant_ip.id
}

# Allocate resources to the tenant
resource "vergeioxp_tenant_node" "primary" {
  provider   = vergeioxp.host
  tenant_id  = vergeioxp_tenant.corp.id
  node_id    = 1
  cpu_cores  = 4
  ram        = 8192
  powerstate = true
}

resource "vergeioxp_tenant_storage" "tier1" {
  provider    = vergeioxp.host
  tenant_id   = vergeioxp_tenant.corp.id
  tier        = 1
  provisioned = 100 * 1024 * 1024 * 1024  # 100GB
}

# --- PHASE 2: Tenant-level resources ---
provider "vergeioxp" {
  alias    = "tenant"
  host     = vergeioxp_vnet_address.tenant_ip.ip  # Dynamically resolved!
  username = "admin"
  password = var.tenant_admin_password
  insecure = true
}

# Upload an image with parallel uploads
resource "vergeioxp_file" "debian_image" {
  provider       = vergeioxp.tenant
  name           = "debian-13.qcow2"
  type           = "qcow"
  source_url     = "https://cloud.debian.org/images/cloud/trixie/latest/debian-13-generic-amd64.qcow2"
  upload_threads = 16  # Fast parallel upload!
  
  depends_on = [vergeioxp_tenant_node.primary]
}

# Create a VM with auto-networking
resource "vergeioxp_vm" "web" {
  provider   = vergeioxp.tenant
  name       = "web-server"
  cpu_cores  = 2
  ram        = 4096
  powerstate = true
  
  # Cloud-init credentials
  username = "admin"
  password = var.vm_password

  vergeio_drive {
    name         = "OS"
    media        = "import"
    media_source = tonumber(vergeioxp_file.debian_image.id)
    interface    = "virtio-scsi"
    disksize     = 20
  }

  vergeio_nic {
    name             = "eth0"
    auto_create_vnet = true  # Creates network, DHCP, and routing!
    vnet_uplink      = 3
  }
}
```

## Provider Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `host` | String | **Yes** | Hostname or IP of the VergeOS system |
| `username` | String | No* | Admin username |
| `password` | String | No* | Admin password (sensitive) |
| `token` | String | No* | API token (alternative to username/password, sensitive) |
| `insecure` | Boolean | No | Skip TLS certificate verification |

*Either `token` OR both `username` and `password` must be provided.

## Resources

- `vergeioxp_vm` — Virtual machines with cloud-init support
- `vergeioxp_vm_snapshot` — VM point-in-time snapshots
- `vergeioxp_tag` — Tag management (full CRUD)
- `vergeioxp_tag_category` — Tag category management
- `vergeioxp_vnet` — Virtual networks
- `vergeioxp_vnet_address` — IP allocations on vnets
- `vergeioxp_tenant` — Multi-tenant environments
- `vergeioxp_tenant_node` — CPU/RAM allocation for tenants
- `vergeioxp_tenant_storage` — Storage allocation for tenants
- `vergeioxp_file` — File uploads with parallel chunked transfer

## Data Sources

- `vergeioxp_vms` — List all VMs
- `vergeioxp_vnets` — List all networks
- `vergeioxp_mediasources` — List available images/ISOs
- `vergeioxp_clusters` — List clusters
- `vergeioxp_nodes` — List physical nodes

## Support

This is an experimental provider. Issues and feature requests can be submitted via GitHub.