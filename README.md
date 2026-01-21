# 🚀 VergeOS Terraform Provider (Experimental)

![VergeOS](https://img.shields.io/badge/VergeOS-Experimental-orange)
![Terraform](https://img.shields.io/badge/Terraform-Provider-blueviolet)

⚠️ **This is an experimental, production-enhanced fork of the official VergeOS Terraform Provider.**

This provider includes advanced features for **Multi-Tenancy**, **Automated Networking**, **Token Authentication**, and **Parallel File Uploads** that are not yet available in the upstream release. It is designed for users who want to treat their entire VergeOS environment—including tenants—as code.

---

## 🌟 Key Features

### 1. 🔐 Token-Based Authentication (NEW!)
Secure API token authentication for CI/CD pipelines:
```hcl
provider "vergeioxp" {
  host     = "192.168.1.111"
  token    = var.verge_token  # No username/password needed!
  insecure = true
}
```

**Create a token via API:**
```bash
curl -sk -X POST "https://your-host/api/sys/tokens" \
  -H "Content-Type: application/json" \
  -d '{"login": "admin", "password": "YourPassword"}'
# Returns: {"location":"/sys/tokens/YOUR_TOKEN_HERE"}
```

### 2. ⚡ Parallel File Uploads (NEW!)
Upload images at maximum speed with configurable thread count:
```hcl
resource "vergeioxp_file" "debian_image" {
  name           = "debian-13.qcow2"
  type           = "qcow"
  source_url     = "https://cloud.debian.org/.../debian-13-generic-amd64.qcow2"
  upload_threads = 16  # 16 parallel upload threads!
}
```

**Performance:** ~414MB file uploads in 20-31 seconds depending on thread count.

### 3. 🏗️ Full Multi-Tenancy Suite
Manage the entire lifecycle of VergeOS Tenants:
- **`vergeioxp_tenant`**: Provision isolated tenant environments with **automatic routing** when linked to a public IP.
- **`vergeioxp_vnet_address`**: Allocate Public IPs from host networks. When linked to a tenant, auto-generates NAT/routing rules.
- **`vergeioxp_tenant_node` & `vergeioxp_tenant_storage`**: Dynamically allocate hardware resources to bring a tenant from "Offline shell" to "Online data center."

### 4. 🏎️ "One-Click" Networking (`auto_create_vnet`)
Stop manually building internal routers. When creating a VM, simply set `auto_create_vnet = true` on the NIC:
- **Instant Overlay**: Automatically provisions a VXLAN network named after the VM.
- **Guided Routing**: Links the new network to an external uplink (`vnet_uplink`) for instant internet access.
- **Self-Service Stack**: Auto-configures DHCP and the internal gateway router.
- **Lifecycle Awareness**: Powers on the network automatically so the VM reaches its first boot successfully.

### 5. 🔑 Zero-Touch Provisioning (Cloud-Init)
Provisioning Linux VMs no longer requires complex `template_file` data sources. Use native fields:
- `username` / `password`: Automatically generates user-data and hashes passwords.
- `ssh_key`: Injects public keys for secure access.
- `hostname`: Sets the OS-level hostname.
- **State Preservation**: These fields are persisted in Terraform state and won't "re-trigger" incorrectly on subsequent runs.

---

## 🛠️ Installation

### 1. Build the Provider
```bash
git clone https://github.com/dvvincent/terraform-provider-vergeioxp
cd terraform-provider-vergeioxp
go build -o terraform-provider-vergeioxp
```

### 2. Configure Development Overrides
Add this to your `~/.terraformrc` (or `%APPDATA%\terraform.rc` on Windows) to point Terraform to your local build:

```hcl
provider_installation {
  dev_overrides {
    "dvvincent/vergeioxp" = "/absolute/path/to/the/built/binary/directory"
  }
  direct {}
}
```

---

## 🌀 Advanced Pattern: "Automation Inception"

The most powerful way to use this provider is the **Bootstrap Pattern**. You use one provider instance (talking to your Host) to create a Tenant and a Public IP, and a **second provider instance** to log into that new Tenant to deploy workloads.

```hcl
# PROVIDER 1: The Host Admin
provider "vergeioxp" {
  alias    = "host"
  host     = "host.vergeos.com"
  username = "admin"
  password = var.host_password
  insecure = true
}

# STEP 1: Create the base infrastructure
resource "vergeioxp_vnet_address" "tenant_ip" {
  provider = vergeioxp.host
  vnet     = 3
  type     = "virtual"
}

resource "vergeioxp_tenant" "corp" {
  provider   = vergeioxp.host
  name       = "customer-a"
  password   = var.tenant_password
  ui_address = vergeioxp_vnet_address.tenant_ip.id
}

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

# PROVIDER 2: The Tenant Admin (Bootstrap)
# This provider waits for the above resources to exist!
provider "vergeioxp" {
  alias    = "tenant"
  host     = vergeioxp_vnet_address.tenant_ip.ip  # Data flow from resource
  username = "admin"
  password = var.tenant_password
  insecure = true
}

# STEP 2: Upload image with parallel upload
resource "vergeioxp_file" "os_image" {
  provider       = vergeioxp.tenant
  name           = "debian-13.qcow2"
  type           = "qcow"
  source_url     = "https://cloud.debian.org/.../debian-13-generic-amd64.qcow2"
  upload_threads = 16
  
  depends_on = [vergeioxp_tenant_node.primary]
}

# STEP 3: Deploy workloads inside the new Tenant
resource "vergeioxp_vm" "web" {
  provider   = vergeioxp.tenant
  name       = "web-server"
  cpu_cores  = 2
  ram        = 4096
  powerstate = true
  
  username = "admin"
  password = "SecurePassword123!"

  vergeio_drive {
    name         = "OS"
    media        = "import"
    media_source = tonumber(vergeioxp_file.os_image.id)
    interface    = "virtio-scsi"
    disksize     = 20
  }

  vergeio_nic {
    name             = "eth0"
    auto_create_vnet = true  # The "Magic" switch
    vnet_uplink      = 3     # Directs traffic to the External network
  }
}
```

---

## 📚 Resource Reference

### Provider Configuration

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `host` | string | **Yes** | VergeOS hostname or IP |
| `username` | string | No* | Admin username |
| `password` | string | No* | Admin password (sensitive) |
| `token` | string | No* | API token (alternative to username/password) |
| `insecure` | bool | No | Skip TLS certificate verification |

*Either `token` OR both `username` and `password` must be provided.

### `vergeioxp_file` (NEW!)

Upload files with parallel chunked transfer.

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | File name in VergeOS |
| `type` | string | File type: `iso`, `qcow`, `img`, `raw` |
| `source_url` | string | URL to download from |
| `upload_threads` | int | Parallel threads (1-24, default 8) |
| `filesize` (computed) | int | Size in bytes after upload |

### `vergeioxp_vnet_rule` (NEW!)

Manage firewall rules and NAT translations.

| Attribute | Type | Description |
|-----------|------|-------------|
| `vnet` | int | Parent Virtual Network ID |
| `name` | string | Rule name |
| `action` | string | `accept`, `drop`, `translate` (NAT) |
| `protocol`| string | `tcp`, `udp`, `icmp`, `any` |
| `destination_ip` | string | Target IP or filter (e.g., `router`) |

### `vergeioxp_vnet_address`

| Attribute | Type | Description |
|-----------|------|-------------|
| `vnet` | int | VNet ID to allocate from (usually 3 for External) |
| `ip` | string | Specific IP address (optional - auto-allocated if omitted) |
| `type` | string | `virtual` (for routing/UI) or `static` |

### `vergeioxp_tenant`

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Tenant name |
| `password` | string | Admin password for the tenant |
| `ui_address` | int | ID of `vergeioxp_vnet_address` for admin portal |
| `vnet_id` (computed) | int | Internal router ID created by VergeOS |

### `vergeioxp_vm` (Enhanced)

| Attribute | Type | Description |
|-----------|------|-------------|
| `username` | string | Cloud-init username |
| `password` | string | Cloud-init password (auto-hashed) |
| `powerstate` | bool | Start VM after creation |
| **vergeio_nic block** | | |
| `auto_create_vnet` | bool | Build isolated internal network |
| `vnet_uplink` | int | External gateway network ID |

---

## ⚠️ Known Limitations & Troubleshooting

- **Tenant Deletion**: VergeOS requires a tenant to be powered off before deletion. If `terraform destroy` fails, ensure the tenant is powered down.
- **API Tokens**: Tokens are session-based and may expire. For long-running automation, consider using username/password or refreshing tokens.
- **VM Snapshots**: The `vergeioxp_vm_snapshot` resource captures CPU and RAM state in addition to disk state.

---

## 🤝 Contributing & Support

This is an experimental fork. If you encounter bugs specific to the new features:
1. Check the [docs/](./docs/) folder for detailed resource documentation.
2. Open an issue on this repository.
3. PRs are welcome!

---
*Built with ❤️ for the VergeOS Community.*
