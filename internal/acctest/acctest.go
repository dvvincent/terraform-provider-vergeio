// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

// Package acctest provides shared utilities for acceptance testing.
package acctest

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"terraform-provider-vergeio/internal/provider"
)

// ProtoV6ProviderFactories is the factory for creating a provider server for testing.
var ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"vergeio": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// PreCheck validates the necessary test API keys exist in the testing environment.
func PreCheck(t *testing.T) {
	t.Helper()

	if v := os.Getenv("VERGEOS_HOST"); v == "" {
		t.Fatal("VERGEOS_HOST must be set for acceptance tests")
	}

	token := os.Getenv("VERGEOS_TOKEN")
	username := os.Getenv("VERGEOS_USERNAME")
	password := os.Getenv("VERGEOS_PASSWORD")

	if token == "" && (username == "" || password == "") {
		t.Fatal("Either VERGEOS_TOKEN or both VERGEOS_USERNAME and VERGEOS_PASSWORD must be set for acceptance tests")
	}
}

// ProviderConfig returns a provider configuration block for acceptance tests.
func ProviderConfig() string {
	host := os.Getenv("VERGEOS_HOST")
	token := os.Getenv("VERGEOS_TOKEN")
	username := os.Getenv("VERGEOS_USERNAME")
	password := os.Getenv("VERGEOS_PASSWORD")

	// Strip protocol prefix if present - the provider adds it internally
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")

	if token != "" {
		return fmt.Sprintf(`
provider "vergeio" {
  host     = %q
  token    = %q
  insecure = true
}
`, host, token)
	}

	return fmt.Sprintf(`
provider "vergeio" {
  host     = %q
  username = %q
  password = %q
  insecure = true
}
`, host, username, password)
}
