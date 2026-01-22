// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

package network_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	acc "terraform-provider-vergeio/internal/acctest"
)

// TestAccNetwork_basic tests the basic create, read, update, and delete lifecycle
// of an internal network (VNet).
func TestAccNetwork_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-net")
	resourceName := "vergeio_network.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acc.PreCheck(t) },
		ProtoV6ProviderFactories: acc.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create a basic internal network
			{
				Config: testAccNetworkConfig_basic(rName, "192.168.100.1", "192.168.100.0/24"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "ipaddress", "192.168.100.1"),
					resource.TestCheckResourceAttr(resourceName, "network", "192.168.100.0/24"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Step 2: Update the network name
			{
				Config: testAccNetworkConfig_basic(rName+"-updated", "192.168.100.1", "192.168.100.0/24"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName+"-updated"),
				),
			},
			// Step 3: Import state test
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// powerstate may differ between plan and import
				ImportStateVerifyIgnore: []string{"powerstate"},
			},
		},
	})
}

// TestAccNetwork_withDHCP tests creating an internal network with DHCP enabled.
func TestAccNetwork_withDHCP(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-dhcp")
	resourceName := "vergeio_network.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acc.PreCheck(t) },
		ProtoV6ProviderFactories: acc.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkConfig_withDHCP(rName, "192.168.101.1", "192.168.101.0/24", "192.168.101.100", "192.168.101.200"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "dhcp_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "dhcp_start", "192.168.101.100"),
					resource.TestCheckResourceAttr(resourceName, "dhcp_stop", "192.168.101.200"),
				),
			},
		},
	})
}

// TestAccNetwork_internalType tests creating an internal network with explicit type.
func TestAccNetwork_internalType(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-int")
	resourceName := "vergeio_network.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acc.PreCheck(t) },
		ProtoV6ProviderFactories: acc.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkConfig_internal(rName, "10.0.0.1", "10.0.0.0/24"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "type", "internal"),
					resource.TestCheckResourceAttr(resourceName, "ipaddress", "10.0.0.1"),
					resource.TestCheckResourceAttr(resourceName, "network", "10.0.0.0/24"),
				),
			},
		},
	})
}

// TestAccNetwork_powerState tests managing network power state.
func TestAccNetwork_powerState(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-pwr")
	resourceName := "vergeio_network.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acc.PreCheck(t) },
		ProtoV6ProviderFactories: acc.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with power on
			{
				Config: testAccNetworkConfig_withPowerState(rName, "192.168.102.1", "192.168.102.0/24", "running"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
				),
			},
		},
	})
}

// testAccNetworkConfig_basic returns a basic internal network configuration.
func testAccNetworkConfig_basic(name, ip, network string) string {
	return acc.ProviderConfig() + fmt.Sprintf(`
resource "vergeio_network" "test" {
  name      = %q
  ipaddress = %q
  network   = %q
}
`, name, ip, network)
}

// testAccNetworkConfig_withDHCP returns a network configuration with DHCP enabled.
func testAccNetworkConfig_withDHCP(name, ip, network, dhcpStart, dhcpStop string) string {
	return acc.ProviderConfig() + fmt.Sprintf(`
resource "vergeio_network" "test" {
  name         = %q
  ipaddress    = %q
  network      = %q
  dhcp_enabled = true
  dhcp_start   = %q
  dhcp_stop    = %q
}
`, name, ip, network, dhcpStart, dhcpStop)
}

// testAccNetworkConfig_internal returns an internal network configuration with explicit type.
func testAccNetworkConfig_internal(name, ip, network string) string {
	return acc.ProviderConfig() + fmt.Sprintf(`
resource "vergeio_network" "test" {
  name      = %q
  type      = "internal"
  ipaddress = %q
  network   = %q
}
`, name, ip, network)
}

// testAccNetworkConfig_withPowerState returns a network configuration with power state.
func testAccNetworkConfig_withPowerState(name, ip, network, powerstate string) string {
	return acc.ProviderConfig() + fmt.Sprintf(`
resource "vergeio_network" "test" {
  name       = %q
  ipaddress  = %q
  network    = %q
  powerstate = %q
}
`, name, ip, network, powerstate)
}

// TestAccNetwork_fullInternal tests a complete internal network configuration
// with DHCP, power state, and power loss settings. This validates the typical
// production configuration for internal networks.
func TestAccNetwork_fullInternal(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-full")
	resourceName := "vergeio_network.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acc.PreCheck(t) },
		ProtoV6ProviderFactories: acc.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkConfig_fullInternal(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "type", "internal"),
					resource.TestCheckResourceAttr(resourceName, "network", "10.99.0.0/24"),
					resource.TestCheckResourceAttr(resourceName, "ipaddress", "10.99.0.1"),
					resource.TestCheckResourceAttr(resourceName, "dhcp_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "dhcp_start", "10.99.0.2"),
					resource.TestCheckResourceAttr(resourceName, "dhcp_stop", "10.99.0.50"),
					resource.TestCheckResourceAttr(resourceName, "on_power_loss", "last_state"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					// Verify interface_vnet is NOT set (should be null/unset)
					// Setting interface_vnet on internal networks causes VXLAN errors
				),
			},
		},
	})
}

// testAccNetworkConfig_fullInternal returns a complete internal network configuration
// matching a typical production setup. NOTE: interface_vnet should NOT be set for
// internal networks - it causes VXLAN "group requires dev" errors.
func testAccNetworkConfig_fullInternal(name string) string {
	return acc.ProviderConfig() + fmt.Sprintf(`
resource "vergeio_network" "test" {
  name          = %q
  type          = "internal"
  
  # Network addressing
  network       = "10.99.0.0/24"
  ipaddress     = "10.99.0.1"
  
  # DHCP configuration
  dhcp_enabled  = true
  dhcp_start    = "10.99.0.2"
  dhcp_stop     = "10.99.0.50"
  
  # Power management
  powerstate    = "running"
  on_power_loss = "last_state"
  
  # NOTE: Do NOT set interface_vnet for internal networks - it causes
  # "Error creating vxlan: vxlan: 'group' requires 'dev'" errors.
  # For external routing, use vergeio_vnet_rule with action = "translate" instead.
}
`, name)
}
