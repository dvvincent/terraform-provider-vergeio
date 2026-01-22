// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

package tags_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	acc "terraform-provider-vergeio/internal/acctest"
)

// TestAccTagCategory_basic tests the basic lifecycle of a tag category.
func TestAccTagCategory_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-cat")
	resourceName := "vergeio_tag_category.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acc.PreCheck(t) },
		ProtoV6ProviderFactories: acc.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccTagCategoryConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name
			{
				Config: testAccTagCategoryConfig_basic(rName + "-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName+"-updated"),
				),
			},
			// Import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccTag_basic tests the basic lifecycle of a tag.
func TestAccTag_basic(t *testing.T) {
	catName := acctest.RandomWithPrefix("tf-acc-cat")
	tagName := acctest.RandomWithPrefix("tf-acc-tag")
	resourceName := "vergeio_tag.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acc.PreCheck(t) },
		ProtoV6ProviderFactories: acc.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create category and tag
			{
				Config: testAccTagConfig_basic(catName, tagName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", tagName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "category"),
				),
			},
			// Update tag name
			{
				Config: testAccTagConfig_basic(catName, tagName+"-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", tagName+"-updated"),
				),
			},
			// Import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// testAccTagCategoryConfig_basic returns a basic tag category configuration.
func testAccTagCategoryConfig_basic(name string) string {
	return acc.ProviderConfig() + fmt.Sprintf(`
resource "vergeio_tag_category" "test" {
  name = %q
}
`, name)
}

// testAccTagConfig_basic returns a tag configuration with its category.
func testAccTagConfig_basic(catName, tagName string) string {
	return acc.ProviderConfig() + fmt.Sprintf(`
resource "vergeio_tag_category" "test" {
  name = %q
}

resource "vergeio_tag" "test" {
  name     = %q
  category = vergeio_tag_category.test.id
}
`, catName, tagName)
}
