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

	"terraform-provider-vergeio/internal/provider/vergeio"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Tenant Storage API endpoint.
const (
	TenantStorageEndpoint = vergeio.APIEndpoint + "/tenant_storage"
)

func NewTenantStorageApi(c *vergeio.Client) *TenantStorageApi {
	return &TenantStorageApi{
		name:   "Tenant Storage Api",
		client: c,
	}
}

type TenantStorageApi struct {
	name   string
	client *vergeio.Client
}

func (tsa *TenantStorageApi) Name() string {
	return tsa.name
}

// TenantStorageAPIModel for creating/updating tenant storage.
type TenantStorageAPIModel struct {
	Tenant      int   `json:"tenant"`
	Tier        int   `json:"tier"`
	Provisioned int64 `json:"provisioned"`
}

// TenantStorageAPIResponseModel for reading tenant storage.
type TenantStorageAPIResponseModel struct {
	Key         int   `json:"$key,omitempty"`
	Tenant      int   `json:"tenant,omitempty"`
	Tier        int   `json:"tier,omitempty"`
	Provisioned int64 `json:"provisioned,omitempty"`
	Used        int64 `json:"used,omitempty"`
	Allocated   int64 `json:"allocated,omitempty"`
}

// Create the tenant storage in the API.
func (tsa *TenantStorageApi) createTenantStorage(ctx context.Context, data *TenantStorageResourceModel) error {
	apiData := TenantStorageAPIModel{
		Tenant:      int(data.TenantID.ValueInt64()),
		Tier:        int(data.Tier.ValueInt64()),
		Provisioned: data.Provisioned.ValueInt64(),
	}

	encodedBuffer := new(bytes.Buffer)
	if err := json.NewEncoder(encodedBuffer).Encode(apiData); err != nil {
		return errors.New("invalid format for tenant storage data")
	}

	tflog.Debug(ctx, fmt.Sprintf("Creating tenant storage with data: %v", encodedBuffer.String()))

	apiResp, err := tsa.client.Post(TenantStorageEndpoint, encodedBuffer)
	if err != nil {
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	if apiResp.StatusCode != 201 {
		return fmt.Errorf("API returned status %d", apiResp.StatusCode)
	}

	var tenantStorageAPIResp vergeio.VergeResponse
	if err := json.NewDecoder(apiResp.Body).Decode(&tenantStorageAPIResp); err != nil {
		return errors.New("invalid format received from API")
	}

	data.Id = types.StringValue(tenantStorageAPIResp.Key)
	tflog.Debug(ctx, fmt.Sprintf("Created tenant storage with Id %v", data.Id.ValueString()))

	return nil
}

// Update the tenant storage in the API.
func (tsa *TenantStorageApi) updateTenantStorage(ctx context.Context, planData *TenantStorageResourceModel, stateData *TenantStorageResourceModel) error {
	apiData := TenantStorageAPIModel{
		Provisioned: planData.Provisioned.ValueInt64(),
	}

	encodedBuffer := new(bytes.Buffer)
	if err := json.NewEncoder(encodedBuffer).Encode(apiData); err != nil {
		return errors.New("invalid format for tenant storage data")
	}

	apiResp, err := tsa.client.Put(fmt.Sprintf("%s/%s",
		TenantStorageEndpoint,
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

// Read the tenant storage from the API.
func (tsa *TenantStorageApi) readTenantStorage(ctx context.Context, data *TenantStorageResourceModel) error {
	apiResp, err := tsa.client.Get(fmt.Sprintf("%s/%s",
		TenantStorageEndpoint,
		url.PathEscape(data.Id.ValueString()),
	), &vergeio.Options{Fields: "$key,tenant,tier,provisioned,used,allocated"})

	if err != nil {
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	if apiResp.StatusCode == 404 {
		return errors.New("tenant storage not found")
	}
	if apiResp.StatusCode != 200 {
		return fmt.Errorf("API returned status %d", apiResp.StatusCode)
	}

	var tenantStorageAPIResp TenantStorageAPIResponseModel
	if err := json.NewDecoder(apiResp.Body).Decode(&tenantStorageAPIResp); err != nil {
		return errors.New("invalid format received from API")
	}

	data.TenantID = types.Int64Value(int64(tenantStorageAPIResp.Tenant))
	data.Tier = types.Int64Value(int64(tenantStorageAPIResp.Tier))
	data.Provisioned = types.Int64Value(tenantStorageAPIResp.Provisioned)
	data.Used = types.Int64Value(tenantStorageAPIResp.Used)
	data.Allocated = types.Int64Value(tenantStorageAPIResp.Allocated)

	return nil
}

// Delete the tenant storage from the API.
func (tsa *TenantStorageApi) deleteTenantStorage(ctx context.Context, data *TenantStorageResourceModel) error {
	_, err := tsa.client.Delete(fmt.Sprintf("%s/%s",
		TenantStorageEndpoint,
		url.PathEscape(data.Id.ValueString())))

	if err != nil {
		return errors.New("error deleting tenant storage: " + err.Error())
	}

	return nil
}
