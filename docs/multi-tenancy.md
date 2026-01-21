# 🏢 Multi-Tenancy Architecture

The `vergeioxp` provider is uniquely designed to handle **Bootstrap Automation**—the process of using Terraform to create a platform, and then using that same platform to deploy workloads.

## 🧱 Core Components

### 1. `vergeioxp_tenant`
Creates the "Shell" of the tenant. By default, a tenant has no resources and is **Offline**.

### 2. `vergeioxp_vnet_address` (The Gateway)
This is the most critical resource for production setups. It allows you to:
- Allocate a **Public IP** from the host's External Network (VNet 3).
- Map that IP to the Tenant UI.
- Trigger automatic DNAT/SNAT rules in VergeOS so the tenant can be reached immediately.

### 3. `vergeioxp_tenant_node` & `vergeioxp_tenant_storage`
These resources "breathe life" into the tenant:
- **Node**: Assigns physical CPU and RAM. Setting `powerstate = true` on the node brings the tenant online.
- **Storage**: Provisions capacity from specific storage tiers.

## 🌀 The Bootstrap Pattern

This is the recommended architecture for deploying new customer environments:

1.  **Phase 1 (Host)**: Call the provider as a Host administrator to create the Tenant and allocate its Public IP.
2.  **Phase 2 (Tenant)**: Call the provider **recursively**, pointing its `host` attribute to the IP address allocated in Phase 1.

```hcl
# This resource creates the IP we need for the next provider
resource "vergeioxp_vnet_address" "tenant_ip" {
  vnet = 3
  ip   = "1.2.3.4"
  type = "virtual"
}

# The INTERNAL provider uses data from the HOST resources
provider "vergeioxp" "customer_api" {
  host     = vergeioxp_vnet_address.tenant_ip.ip
  username = "admin"
  password = var.customer_password
}

# Now we can deploy VMs INSIDE the new customer environment
resource "vergeioxp_vm" "customer_workload" {
  provider = vergeioxp.customer_api
  name     = "primary-vm"
  # ...
}
```

## 🔗 Automatic Routing (ui_address Magic)

When you link a `vergeioxp_vnet_address` to a tenant via the `ui_address` attribute, the provider automatically:
1. Sets the `owner` field on the vnet_address to `tenants/{id}`
2. This triggers VergeOS to auto-generate the **routing rules** on the External network
3. The tenant becomes immediately accessible via its Public IP

This eliminates the need to manually create DNAT/SNAT rules—it's all handled automatically!

## ⚠️ Important: Deletion Order

VergeOS requires a specific shutdown sequence before a tenant can be deleted:

1. **Power off the Tenant Node**: Set `powerstate = false` on `vergeioxp_tenant_node` and run `terraform apply`
2. **Power off the Tenant Network** (via API or UI):
   ```bash
   curl -X POST "https://host/api/v4/vnet_actions" \
     -d '{"vnet": TENANT_VNET_ID, "action": "kill", "params": {}}'
   ```
3. **Run `terraform destroy`**

> **Note:** Future versions may automate this shutdown sequence during destroy operations.

