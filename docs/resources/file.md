---
page_title: "vergeioxp_file Resource - VergeIO Experimental Provider"
subcategory: ""
description: |-
  Manages file uploads in VergeOS with parallel chunked transfer.
---

# vergeioxp_file (Resource)

The `vergeioxp_file` resource allows you to upload files (ISO images, QCOW2 disk images, etc.) to VergeOS. Files are uploaded using a parallel chunked transfer mechanism for maximum performance.

## Features

- **Parallel chunked uploads** — Files are split into 256KB chunks and uploaded in parallel
- **Configurable thread count** — Control upload parallelism with `upload_threads` (1-24)
- **URL-based downloads** — Specify a `source_url` and the provider downloads and uploads automatically
- **Progress reporting** — Upload progress is logged during Terraform apply

## Example Usage

### Basic Image Upload

```hcl
resource "vergeioxp_file" "debian_image" {
  name       = "debian-13.qcow2"
  type       = "qcow"
  source_url = "https://cloud.debian.org/images/cloud/trixie/latest/debian-13-generic-amd64.qcow2"
}
```

### High-Performance Upload with 16 Threads

```hcl
resource "vergeioxp_file" "windows_iso" {
  name           = "windows-server-2022.iso"
  type           = "iso"
  source_url     = "https://example.com/windows-2022.iso"
  upload_threads = 16  # Use 16 parallel upload threads
}
```

### Use with VM Resource

```hcl
resource "vergeioxp_file" "os_image" {
  name           = "ubuntu-24.04.qcow2"
  type           = "qcow"
  source_url     = "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img"
  upload_threads = 8
}

resource "vergeioxp_vm" "server" {
  name      = "my-server"
  cpu_cores = 2
  ram       = 4096

  vergeio_drive {
    name         = "OS"
    media        = "import"
    media_source = tonumber(vergeioxp_file.os_image.id)
    interface    = "virtio-scsi"
    disksize     = 40
  }
}
```

## Argument Reference

The following arguments are supported:

- `name` - (Required) The name of the file in VergeOS.

- `type` - (Required) The type of file. Valid values:
  - `iso` — ISO disk image
  - `qcow` — QCOW2 disk image
  - `img` — Raw disk image
  - `raw` — Raw disk image

- `source_url` - (Required) The URL to download the file from. The file is downloaded to a temporary location and then uploaded to VergeOS using chunked transfer. Changing this value forces recreation of the resource.

- `description` - (Optional) A description for the file.

- `upload_threads` - (Optional) Number of parallel upload threads. Valid range: 1-24. Default: 8.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

- `id` - The unique identifier of the file in VergeOS.

- `filesize` - The size of the file in bytes.

## Import

Files can be imported using their ID:

```shell
terraform import vergeioxp_file.example 123
```

## Performance Notes

- **Default threads (8)** — Good balance for most networks
- **16 threads** — Recommended for fast local networks
- **24 threads (max)** — Maximum parallelism, may stress network

A 414MB file uploads in approximately:
- ~31 seconds with 8 threads
- ~20 seconds with 16 threads

## Implementation Details

The file upload uses the same chunked transfer mechanism as the official `verge-cli` tool:

1. File is downloaded from `source_url` to a temporary file
2. File is registered in VergeOS with `allocated_bytes` pre-allocation
3. File is uploaded in 256KB chunks using `PUT /api/v4/files/{id}?filepos={offset}`
4. Each chunk uses a fresh HTTP connection (`DisableKeepAlives: true`)
5. Chunks are uploaded in parallel using goroutines controlled by a semaphore
