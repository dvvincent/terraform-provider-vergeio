---
page_title: "vergeioxp_vm_snapshot Resource - VergeIO Experimental Provider"
subcategory: ""
description: |-
  Manages a VM snapshot in VergeOS.
---

# vergeioxp_vm_snapshot (Resource)

The `vergeioxp_vm_snapshot` resource allows you to create and manage snapshots of Virtual Machines in VergeOS.

## Features

- **Point-in-time state** — Capture the exact state of a VM's disk and configuration.
- **Independent lifecycle** — Manage snapshots separately from the VM itself.
- **Fast creation** — Leverages VergeOS global deduplication for instant snapshots.

## Example Usage

### Basic Snapshot

```hcl
resource "vergeioxp_vm" "web" {
  name      = "web-server"
  cpu_cores = 2
  ram       = 4096
  # ... other config ...
}

resource "vergeioxp_vm_snapshot" "pre_upgrade" {
  vm_id       = vergeioxp_vm.web.id
  name        = "pre-upgrade-snap"
  description = "Snapshot taken before OS upgrade"
}
```

## Argument Reference

The following arguments are supported:

- `vm_id` - (Required) The ID of the VM to snapshot. Changing this forces a new resource.

- `name` - (Required) The name of the snapshot.

- `description` - (Optional) A description for the snapshot.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

- `id` - The unique identifier of the snapshot.

- `machine_id` - The internal machine ID of the parent VM.

- `snap_machine_id` - The ID of the resulting "frozen" machine created by the snapshot.

- `created` - Unix timestamp when the snapshot was created.

## Import

Snapshots can be imported using their ID:

```shell
terraform import vergeioxp_vm_snapshot.example 123
```
