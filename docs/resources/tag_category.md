---
page_title: "vergeioxp_tag_category Resource - VergeIO Experimental Provider"
subcategory: ""
description: |-
  Manages a tag category in VergeOS.
---

# vergeioxp_tag_category (Resource)

The `vergeioxp_tag_category` resource allows you to create and manage tag categories in VergeOS. Tag categories are containers that organize tags and define which resource types can be tagged.

## Features

- **Full CRUD** — Create, Read, Update, and Delete tag categories programmatically.
- **Resource Type Control** — Define which types of resources (VMs, Tenants, Networks, etc.) can be tagged.
- **Single Selection Mode** — Optionally restrict resources to one tag from this category.

## Example Usage

### Basic Tag Category with Tags

```hcl
# Create a tag category for infrastructure classification
resource "vergeioxp_tag_category" "environment" {
  name                 = "Environment"
  description          = "Classify resources by deployment environment"
  taggable_vms         = true
  taggable_tenants     = true
  taggable_vnets       = true
  single_tag_selection = true  # Only one tag from this category per resource
}

# Create tags within the category
resource "vergeioxp_tag" "production" {
  name        = "production"
  description = "Production environment"
  category    = vergeioxp_tag_category.environment.id
}

resource "vergeioxp_tag" "staging" {
  name        = "staging"
  description = "Staging environment"
  category    = vergeioxp_tag_category.environment.id
}

resource "vergeioxp_tag" "development" {
  name        = "development"
  description = "Development environment"
  category    = vergeioxp_tag_category.environment.id
}
```

### Tagging VMs

```hcl
resource "vergeioxp_vm" "web_server" {
  name      = "web-server-prod"
  cpu_cores = 4
  ram       = 8192
  # ... other config ...
}

resource "vergeioxp_tag_member" "web_is_prod" {
  tag_id = vergeioxp_tag.production.id
  member = "vms/${vergeioxp_vm.web_server.id}"
}
```

## Argument Reference

The following arguments are supported:

- `name` - (Required) The name of the tag category.

- `description` - (Optional) A description of the tag category.

- `single_tag_selection` - (Optional, default: `false`) If true, only one tag from this category can be assigned to a resource at a time.

- `taggable_vms` - (Optional, default: `true`) Allow tagging VMs with tags from this category.

- `taggable_volumes` - (Optional, default: `false`) Allow tagging Volumes.

- `taggable_vnets` - (Optional, default: `false`) Allow tagging Virtual Networks.

- `taggable_vnet_rules` - (Optional, default: `false`) Allow tagging VNet Rules.

- `taggable_tenants` - (Optional, default: `false`) Allow tagging Tenants.

- `taggable_tenant_nodes` - (Optional, default: `false`) Allow tagging Tenant Nodes.

- `taggable_users` - (Optional, default: `false`) Allow tagging Users.

- `taggable_nodes` - (Optional, default: `false`) Allow tagging physical Nodes.

- `taggable_clusters` - (Optional, default: `false`) Allow tagging Clusters.

- `taggable_groups` - (Optional, default: `false`) Allow tagging Groups.

- `taggable_sites` - (Optional, default: `false`) Allow tagging Sites.

- `taggable_vmware_containers` - (Optional, default: `false`) Allow tagging VMware Containers.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

- `id` - The unique identifier of the tag category.

## Import

Tag categories can be imported using their ID:

```shell
terraform import vergeioxp_tag_category.example 123
```

## Notes

- **Delete Behavior**: Deleting a category will fail if it still contains tags. Delete all tags in a category first.
- **Used with Tags**: Create categories first, then use them with `vergeioxp_tag` resources.
