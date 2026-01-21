# `vergeioxp_vnet_rule` (Resource)

Manages a Virtual Network (vNET) firewall rule or NAT translation.

## Example Usage

### Basic Firewall Rule (Accept HTTP)
```hcl
resource "vergeioxp_vnet_rule" "allow_http" {
  vnet              = 6
  name              = "Allow HTTP"
  description       = "Allow incoming HTTP traffic"
  protocol          = "tcp"
  direction         = "incoming"
  action            = "accept"
  destination_ip    = "192.168.0.241"
  destination_ports = "80"
}
```

### NAT Translation (Port Forwarding)
```hcl
resource "vergeioxp_vnet_rule" "web_nat" {
  vnet              = 6
  name              = "Web NAT"
  action            = "translate"
  destination_ip    = "router"
  destination_ports = "8080"
  target_ip         = "192.168.0.241"
  target_ports      = "80"
}
```

## Schema

### Required
- `vnet` (Number) The ID of the virtual network this rule belongs to.

### Optional
- `name` (String) Name of the rule.
- `description` (String) Description of the rule.
- `enabled` (Boolean) Whether the rule is enabled. Defaults to `true`.
- `orderid` (Number) Ordering/Priority of the rule. If not specified, VergeOS assigns the next available increment.
- `protocol` (String) Protocol (tcp, udp, icmp, any, etc). Defaults to `any`.
- `direction` (String) Traffic direction (incoming, outgoing). Defaults to `incoming`.
- `interface` (String) Interface to apply the rule on (auto, external, internal). Defaults to `auto`.
- `action` (String) Rule action (accept, drop, reject, translate, etc). Defaults to `accept`.
- `source_ip` (String) Source IP or smart filter.
- `source_ports` (String) Source port or range.
- `destination_ip` (String) Destination IP or smart filter.
- `destination_ports` (String) Destination port or range.
- `target_ip` (String) Target IP for NAT translation.
- `target_ports` (String) Target port for NAT translation.

### Read-Only
- `id` (String) The ID of the rule.

## Notes
- **Automatic Refresh**: Every change to a vNET rule automatically triggers a `refresh` action on the parent Virtual Network to apply the firewall changes immediately.
- **Smart Filters**: `destination_ip` and `source_ip` support VergeOS smart filters like `router` or `vmnic:ID.NIC`. (Note: use raw IPs if you encounter syntax errors in VergeOS logs).
