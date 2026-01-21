# 🌐 Deep Dive: Automated Networking (`auto_create_vnet`)

One of the most complex manual tasks in VergeOS is provisioning a new network for a specific workload. This involves creating the VNet, configuring a router, setting up DHCP, and bridging it to an uplink. The `vergeioxp` provider automates this entire stack.

## 🛠️ How it Works

When you set `auto_create_vnet = true` on a `vergeio_nic` block, the provider executes the following sequence during the `Create` operation:

### 1. Identify the Uplink
The `vnet_uplink` attribute specifies which network (e.g., VNet 3 for External) should provide internet access or routing to the new network.

### 2. Provision the Overlay (VXLAN)
The provider creates a new **Internal Network** (Type: `internal`). 
- **Name**: Inherits the VM's name for easy identification.
- **Layer 2**: Automatically uses **VXLAN** for the transport layer.
- **Physical Mesh**: Automatically identifies the physical interface mesh to use for the overlay, so you don't have to specify `interface_vnet`.

### 3. Orchestrate Routing
The provider configures the **Virtual Router** for this new network:
- **Default Gateway**: Sets `vnet_default_gateway` to point to your `vnet_uplink`.
- **DHCP**: Enables DHCP and Dynamic allocation (range `.2` to `.254`) automatically.
- **DNS**: Points DNS to the uplink's DNS configuration.

### 4. Lifecycle Synchronization
Usually, a VM booting before its network is "Ready" results in a failed DHCP lease. The provider ensures the network is **powered on** via the VergeOS API *immediately* after creation and *before* returning control to the VM resource.

## 📊 Example Configuration

```hcl
resource "vergeioxp_vm" "isolated_server" {
  name = "app-db-1"

  vergeio_nic {
    name             = "eth0"
    auto_create_vnet = true # Builds the 'app-db-1' network
    vnet_uplink      = 3    # Routes through External network
  }
}
```

## ⚠️ Requirements
- **Cloud-Init/DHCP**: The guest OS must be configured to use DHCP on the interface.
- **Uplink**: The `vnet_uplink` must be a valid, running network with routing capabilities (Core, DMZ, or External).
