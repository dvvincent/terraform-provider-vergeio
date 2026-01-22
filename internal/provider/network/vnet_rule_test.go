// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

package network_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	acc "terraform-provider-vergeio/internal/acctest"
)

// TestAccVNetRule_basic tests the basic create, read, update, and delete lifecycle
// of a VNet firewall rule.
func TestAccVNetRule_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "vergeio_vnet_rule.test"

	// Get test VNet ID from environment or use a default
	vnetID := os.Getenv("VERGEOS_TEST_VNET_ID")
	if vnetID == "" {
		t.Skip("VERGEOS_TEST_VNET_ID must be set for VNet rule acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acc.PreCheck(t) },
		ProtoV6ProviderFactories: acc.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create a basic accept rule
			{
				Config: testAccVNetRuleConfig_basic(rName, vnetID, "80"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "action", "accept"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "tcp"),
					resource.TestCheckResourceAttr(resourceName, "direction", "incoming"),
					resource.TestCheckResourceAttr(resourceName, "destination_ports", "80"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Step 2: Update the destination port (tests update lifecycle)
			{
				Config: testAccVNetRuleConfig_basic(rName, vnetID, "443"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "destination_ports", "443"),
				),
			},
			// Step 3: Import state test
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccVNetRule_nat tests creation of a NAT (translate) rule.
func TestAccVNetRule_nat(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-nat")
	resourceName := "vergeio_vnet_rule.test"

	vnetID := os.Getenv("VERGEOS_TEST_VNET_ID")
	if vnetID == "" {
		t.Skip("VERGEOS_TEST_VNET_ID must be set for VNet rule acceptance tests")
	}

	targetIP := os.Getenv("VERGEOS_TEST_TARGET_IP")
	if targetIP == "" {
		targetIP = "192.168.0.100" // Default test IP
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acc.PreCheck(t) },
		ProtoV6ProviderFactories: acc.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a NAT rule
			{
				Config: testAccVNetRuleConfig_nat(rName, vnetID, targetIP, "8080", "80"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "action", "translate"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "tcp"),
					resource.TestCheckResourceAttr(resourceName, "destination_ports", "8080"),
					resource.TestCheckResourceAttr(resourceName, "target_ip", targetIP),
					resource.TestCheckResourceAttr(resourceName, "target_ports", "80"),
				),
			},
		},
	})
}

// TestAccVNetRule_drop tests creation of a drop rule.
func TestAccVNetRule_drop(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-drop")
	resourceName := "vergeio_vnet_rule.test"

	vnetID := os.Getenv("VERGEOS_TEST_VNET_ID")
	if vnetID == "" {
		t.Skip("VERGEOS_TEST_VNET_ID must be set for VNet rule acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acc.PreCheck(t) },
		ProtoV6ProviderFactories: acc.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVNetRuleConfig_drop(rName, vnetID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "action", "drop"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "icmp"),
					resource.TestCheckResourceAttr(resourceName, "direction", "incoming"),
				),
			},
		},
	})
}

// testAccVNetRuleConfig_basic returns a basic accept rule configuration.
func testAccVNetRuleConfig_basic(name, vnetID, port string) string {
	return acc.ProviderConfig() + fmt.Sprintf(`
resource "vergeio_vnet_rule" "test" {
  vnet              = %s
  name              = %q
  action            = "accept"
  protocol          = "tcp"
  direction         = "incoming"
  destination_ports = %q
  description       = "Acceptance test rule: %s"
  enabled           = true
}
`, vnetID, name, port, name)
}

// testAccVNetRuleConfig_nat returns a NAT (translate) rule configuration.
func testAccVNetRuleConfig_nat(name, vnetID, targetIP, srcPort, dstPort string) string {
	return acc.ProviderConfig() + fmt.Sprintf(`
resource "vergeio_vnet_rule" "test" {
  vnet              = %s
  name              = %q
  action            = "translate"
  protocol          = "tcp"
  direction         = "incoming"
  destination_ports = %q
  target_ip         = %q
  target_ports      = %q
  description       = "NAT test rule: %s"
  enabled           = true
}
`, vnetID, name, srcPort, targetIP, dstPort, name)
}

// testAccVNetRuleConfig_drop returns a drop rule configuration.
func testAccVNetRuleConfig_drop(name, vnetID string) string {
	return acc.ProviderConfig() + fmt.Sprintf(`
resource "vergeio_vnet_rule" "test" {
  vnet        = %s
  name        = %q
  action      = "drop"
  protocol    = "icmp"
  direction   = "incoming"
  description = "Drop ICMP test: %s"
  enabled     = true
}
`, vnetID, name, name)
}
