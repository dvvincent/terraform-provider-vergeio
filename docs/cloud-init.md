# 🔑 Managed Cloud-Init & Credentials

Provisioning Linux VMs often requires creating repetitive `cloud-init` templates. The `vergeioxp` provider introduces high-level fields that simplify this process while maintaining security.

## 🚀 Native Fields

The `vergeioxp_vm` resource now supports these top-level attributes:

| Field | Description |
|-------|-------------|
| `username` | The default user to create (e.g., `admin`, `debian`). |
| `password` | The user's password. This is **auto-hashed** by the provider before submission. |
| `ssh_key` | Public SSH key to inject into `authorized_keys`. |
| `hostname` | Sets the machine hostname in the OS. |

## 🛡️ Hashing and Security

VergeOS expects Cloud-Init passwords to be hashed. If you provide a password in plain text, the provider:
1. Detects it's a new password.
2. Uses the `plain_text_passwd` VergeOS API field during creation.
3. VergeOS then hashes the password securely using the `nocloud` datasource.

## 🔄 State Persistence (The "Official Provider Fix")

A common issue in the official VergeOS provider is "flapping" cloud-init state. Because the VergeOS API doesn't always return the `cloudinit_files` contents after they are set, Terraform thinks they have been deleted and tries to re-apply them on every run.

The `vergeioxp` provider implements **Smart Persistence**:
- It stores the original `user-data` and `meta-data` in the Terraform state.
- During a `Read` operation, if the API returns an empty list for `cloudinit_files`, the provider **re-injects the values from the state**.
- This prevents unnecessary VM updates and keep your `terraform plan` clean.

## 🛠️ Example

```hcl
resource "vergeioxp_vm" "secure_vm" {
  name     = "my-server"
  username = "ubuntu"
  password = "VerySecretPassword!"
  ssh_key  = "ssh-rsa AAAA..."
  hostname = "web-prod-1"
  
  # The provider automatically generates the cloud-init config!
}
```

## ⚠️ Notes
- **First Boot Only**: Cloud-init only applies these settings on the **first boot**. Changing the password in Terraform *after* the VM is already running will not change the password in the OS unless you recreate the VM.
- **Datasource**: This feature forces the `cloudinit_datasource` to `nocloud`.
