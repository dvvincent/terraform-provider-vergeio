---
page_title: "vergeioxp_tag Resource - VergeIO Experimental Provider"
subcategory: ""
description: |-
  Manages a tag in VergeOS.
---

# vergeioxp_tag (Resource)

The `vergeioxp_tag` resource allows you to create and manage tags in VergeOS. Tags are used to organize and categorize resources like VMs, networks, and volumes.

## Features

- **Full CRUD** — Create, Read, Update, and Delete tags programmatically.
- **Categorization** — Tags belong to categories for better organization.
- **Resource Tagging** — Once created, use `vergeioxp_tag_member` to assign tags to resources.

## Example Usage

### Basic Tag

```hcl
# First, you need a tag category (can be created via the UI or API)
# Category ID 1 is assumed to exist

resource "vergeioxp_tag" "terraform_managed" {
  name        = "terraform-managed"
  description = "Resources managed by Terraform IaC"
  category    = 1
}

resource "vergeioxp_tag" "production" {
  name        = "production"
  description = "Production workloads"
  category    = 1
}
```

### Tagging a VM

```hcl
resource "vergeioxp_tag" "web_tier" {
  name        = "web-tier"
  description = "Web-facing servers"
  category    = 1
}

resource "vergeioxp_vm" "web_server" {
  name      = "web-server-1"
  cpu_cores = 2
  ram       = 4096
  # ... other config ...
}

resource "vergeioxp_tag_member" "web_server_tag" {
  tag_id = vergeioxp_tag.web_tier.id
  member = "vms/${vergeioxp_vm.web_server.id}"
}
```

## Argument Reference

The following arguments are supported:

- `name` - (Required) The name of the tag.

- `category` - (Required) The ID of the tag category this tag belongs to. Tag categories must be created before tags can be assigned to them.

- `description` - (Optional) A description for the tag.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

- `id` - The unique identifier of the tag.

## Import

Tags can be imported using their ID:

```shell
terraform import vergeioxp_tag.example 123
```

## Notes

- **Categories are Required**: VergeOS requires all tags to belong to a category. You must create a tag category first (via the VergeOS UI or directly via the API) before you can create tags.
- **Tag Members**: To assign a tag to a resource (like a VM), use the `vergeioxp_tag_member` resource.
