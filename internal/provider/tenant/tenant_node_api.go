// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

package tenant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"terraform-provider-vergeio/internal/provider/vergeio"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Tenant Node API endpoint.
const (
	TenantNodeEndpoint = vergeio.APIEndpoint + "/tenant_nodes"
)

func NewTenantNodeApi(c *vergeio.Client) *TenantNodeApi {
	return &TenantNodeApi{
		name:   "Tenant Node Api",
		client: c,
	}
}

type TenantNodeApi struct {
	name   string
	client *vergeio.Client
}

func (tna *TenantNodeApi) Name() string {
	return tna.name
}

// TenantNodeAPIModel for creating/updating tenant nodes.
type TenantNodeAPIModel struct {
	Tenant       int    `json:"tenant"`
	NodeID       int    `json:"nodeid,omitempty"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	Enabled      bool   `json:"enabled"`
	Powerstate   bool   `json:"powerstate"`
	CPUCores     int    `json:"cpu_cores"`
	RAM          int    `json:"ram"`
	OnPowerLoss  string `json:"on_power_loss,omitempty"`
}

// TenantNodeAPIResponseModel for reading tenant nodes.
type TenantNodeAPIResponseModel struct {
	Key          int    `json:"$key,omitempty"`
	Tenant       int    `json:"tenant,omitempty"`
	NodeID       int    `json:"nodeid,omitempty"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	Enabled      bool   `json:"enabled"`
	Powerstate   bool   `json:"powerstate"`
	CPUCores     int    `json:"cpu_cores,omitempty"`
	RAM          int    `json:"ram,omitempty"`
	OnPowerLoss  string `json:"on_power_loss,omitempty"`
}

// TenantNodeAPIUpdateModel for updating tenant nodes (excludes readonly fields).
type TenantNodeAPIUpdateModel struct {
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	Enabled      bool   `json:"enabled"`
	Powerstate   bool   `json:"powerstate"`
	CPUCores     int    `json:"cpu_cores"`
	RAM          int    `json:"ram"`
	OnPowerLoss  string `json:"on_power_loss,omitempty"`
}

// Create the tenant node in the API.
func (tna *TenantNodeApi) createTenantNode(ctx context.Context, data *TenantNodeResourceModel) error {
	apiData := TenantNodeAPIModel{
		Tenant:      int(data.TenantID.ValueInt64()),
		NodeID:      int(data.NodeID.ValueInt64()),
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Enabled:     data.Enabled.ValueBool(),
		Powerstate:  data.Powerstate.ValueBool(),
		CPUCores:    int(data.CPUCores.ValueInt64()),
		RAM:         int(data.RAM.ValueInt64()),
		OnPowerLoss: data.OnPowerLoss.ValueString(),
	}

	encodedBuffer := new(bytes.Buffer)
	if err := json.NewEncoder(encodedBuffer).Encode(apiData); err != nil {
		return errors.New("invalid format for tenant node data")
	}

	tflog.Debug(ctx, fmt.Sprintf("Creating tenant node with data: %v", encodedBuffer.String()))

	apiResp, err := tna.client.Post(TenantNodeEndpoint, encodedBuffer)
	if err != nil {
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	if apiResp.StatusCode != 201 {
		return fmt.Errorf("API returned status %d", apiResp.StatusCode)
	}

	var tenantNodeAPIResp vergeio.VergeResponse
	if err := json.NewDecoder(apiResp.Body).Decode(&tenantNodeAPIResp); err != nil {
		return errors.New("invalid format received from API")
	}

	data.Id = types.StringValue(tenantNodeAPIResp.Key)
	tflog.Debug(ctx, fmt.Sprintf("Created tenant node with Id %v", data.Id.ValueString()))

	return nil
}

// Update the tenant node in the API.
func (tna *TenantNodeApi) updateTenantNode(ctx context.Context, planData *TenantNodeResourceModel, stateData *TenantNodeResourceModel) error {
	apiData := TenantNodeAPIUpdateModel{
		Name:        planData.Name.ValueString(),
		Description: planData.Description.ValueString(),
		Enabled:     planData.Enabled.ValueBool(),
		Powerstate:  planData.Powerstate.ValueBool(),
		CPUCores:    int(planData.CPUCores.ValueInt64()),
		RAM:         int(planData.RAM.ValueInt64()),
		OnPowerLoss: planData.OnPowerLoss.ValueString(),
	}

	encodedBuffer := new(bytes.Buffer)
	if err := json.NewEncoder(encodedBuffer).Encode(apiData); err != nil {
		return errors.New("invalid format for tenant node data")
	}

	apiResp, err := tna.client.Put(fmt.Sprintf("%s/%s",
		TenantNodeEndpoint,
		url.PathEscape(stateData.Id.ValueString()),
	), encodedBuffer)
	if err != nil {
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	if apiResp.StatusCode != 200 {
		return fmt.Errorf("API returned status %d", apiResp.StatusCode)
	}

	defer apiResp.Body.Close()
	return nil
}

// Read the tenant node from the API.
func (tna *TenantNodeApi) readTenantNode(ctx context.Context, data *TenantNodeResourceModel) error {
	apiResp, err := tna.client.Get(fmt.Sprintf("%s/%s",
		TenantNodeEndpoint,
		url.PathEscape(data.Id.ValueString()),
	), &vergeio.Options{Fields: "$key,tenant,nodeid,name,description,enabled,machine#status#powerstate as powerstate,cpu_cores,ram,on_power_loss"})

	if err != nil {
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	if apiResp.StatusCode == 404 {
		return errors.New("tenant node not found")
	}
	if apiResp.StatusCode != 200 {
		return fmt.Errorf("API returned status %d", apiResp.StatusCode)
	}

	var tenantNodeAPIResp TenantNodeAPIResponseModel
	if err := json.NewDecoder(apiResp.Body).Decode(&tenantNodeAPIResp); err != nil {
		return errors.New("invalid format received from API")
	}

	data.TenantID = types.Int64Value(int64(tenantNodeAPIResp.Tenant))
	data.NodeID = types.Int64Value(int64(tenantNodeAPIResp.NodeID))
	data.Name = types.StringValue(tenantNodeAPIResp.Name)
	data.Description = types.StringValue(tenantNodeAPIResp.Description)
	data.Enabled = types.BoolValue(tenantNodeAPIResp.Enabled)
	data.Powerstate = types.BoolValue(tenantNodeAPIResp.Powerstate)
	data.CPUCores = types.Int64Value(int64(tenantNodeAPIResp.CPUCores))
	data.RAM = types.Int64Value(int64(tenantNodeAPIResp.RAM))
	data.OnPowerLoss = types.StringValue(tenantNodeAPIResp.OnPowerLoss)

	return nil
}

// Delete the tenant node from the API.
func (tna *TenantNodeApi) deleteTenantNode(ctx context.Context, data *TenantNodeResourceModel) error {
	tflog.Debug(ctx, fmt.Sprintf("Deleting tenant node %s", data.Id.ValueString()))

	// First, power off the tenant node if it's running
	// We need to set powerstate to false
	powerOffPayload := map[string]interface{}{
		"powerstate": false,
	}
	encodedBuffer := new(bytes.Buffer)
	if err := json.NewEncoder(encodedBuffer).Encode(powerOffPayload); err != nil {
		return errors.New("error encoding power off payload")
	}

	// Send the power off request
	_, err := tna.client.Put(fmt.Sprintf("%s/%s",
		TenantNodeEndpoint,
		url.PathEscape(data.Id.ValueString()),
	), encodedBuffer)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Warning: Could not power off tenant node: %v", err))
	}

	// Wait for the tenant node to power off (poll up to 30 seconds)
	for i := 0; i < 6; i++ {
		time.Sleep(5 * time.Second)

		// Check the powerstate
		apiResp, err := tna.client.Get(fmt.Sprintf("%s/%s",
			TenantNodeEndpoint,
			url.PathEscape(data.Id.ValueString()),
		), &vergeio.Options{Fields: "machine#status#powerstate as powerstate"})
		if err != nil {
			break // Node may already be gone
		}
		if apiResp == nil || apiResp.StatusCode != 200 {
			break
		}

		var statusResp struct {
			Powerstate bool `json:"powerstate"`
		}
		if err := json.NewDecoder(apiResp.Body).Decode(&statusResp); err != nil {
			break
		}
		if !statusResp.Powerstate {
			tflog.Debug(ctx, "Tenant node powered off successfully")
			break
		}
		tflog.Debug(ctx, fmt.Sprintf("Waiting for tenant node to power off... attempt %d", i+1))
	}

	// Now delete the tenant node
	_, err = tna.client.Delete(fmt.Sprintf("%s/%s",
		TenantNodeEndpoint,
		url.PathEscape(data.Id.ValueString())))

	if err != nil {
		return errors.New("error deleting tenant node: " + err.Error())
	}

	tflog.Debug(ctx, "Tenant node was successfully deleted")
	return nil
}
