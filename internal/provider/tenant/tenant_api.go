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

// Tenant API endpoint.
const (
	TenantEndpoint = vergeio.APIEndpoint + "/tenants"
)

// IClient interface.
var _ vergeio.IClient = &TenantApi{}

func NewTenantApi(c *vergeio.Client) *TenantApi {
	return &TenantApi{
		name:   "Tenant Api",
		client: c,
	}
}

type TenantApi struct {
	name   string
	client *vergeio.Client
}

func (tc *TenantApi) Name() string {
	return tc.name
}

// TenantAPIResourceModel describes the data model received from the Verge API.
type TenantAPIResourceModel struct {
	Key                   string `json:"$key,omitempty"`
	Name                  string `json:"name,omitempty"`
	Description           string `json:"description,omitempty"`
	Password              string `json:"password,omitempty"`
	ChangePassword        bool   `json:"change_password,omitempty"`
	ExposeCloudSnapshots  bool   `json:"expose_cloud_snapshots,omitempty"`
	AllowBranding         bool   `json:"allow_branding,omitempty"`
	URL                   string `json:"url,omitempty"`
	Note                  string `json:"note,omitempty"`
	Isolate               bool   `json:"isolate,omitempty"`
	ThemeAccess           string `json:"theme_access,omitempty"`
	HelpURL               string `json:"help_url,omitempty"`
	UIAddress             string `json:"ui_address,omitempty"`
	UIFQDN                string `json:"ui_fqdn,omitempty"`
	UUID                  string `json:"uuid,omitempty"`
}

// TenantAPIResponseModel for reading tenants (includes computed fields).
type TenantAPIResponseModel struct {
	Key                   int     `json:"$key,omitempty"`
	Name                  string  `json:"name,omitempty"`
	Description           string  `json:"description,omitempty"`
	ExposeCloudSnapshots  bool    `json:"expose_cloud_snapshots"`
	AllowBranding         bool    `json:"allow_branding"`
	URL                   string  `json:"url,omitempty"`
	Note                  string  `json:"note,omitempty"`
	Isolate               bool    `json:"isolate"`
	ThemeAccess           string  `json:"theme_access,omitempty"`
	HelpURL               string      `json:"help_url,omitempty"`
	UIAddress             interface{} `json:"ui_address"`
	UIFQDN                interface{} `json:"ui_fqdn"`
	UUID                  string  `json:"uuid,omitempty"`
	Created               int64   `json:"created,omitempty"`
	Modified              int64   `json:"modified,omitempty"`
	Owner                 *string `json:"owner"`
	Creator               string  `json:"creator,omitempty"`
	VNet                  int     `json:"vnet,omitempty"`
}

// Create the tenant in the API.
func (tc *TenantApi) createTenant(ctx context.Context, data *TenantResourceModel) error {

	apiData := TenantAPIResourceModel{
		Name:                  data.Name.ValueString(),
		Description:           data.Description.ValueString(),
		Password:              data.Password.ValueString(),
		ChangePassword:        data.ChangePassword.ValueBool(),
		ExposeCloudSnapshots:  data.ExposeCloudSnapshots.ValueBool(),
		AllowBranding:         data.AllowBranding.ValueBool(),
		URL:                   data.URL.ValueString(),
		Note:                  data.Note.ValueString(),
		Isolate:               data.Isolate.ValueBool(),
		ThemeAccess:           data.ThemeAccess.ValueString(),
		HelpURL:               data.HelpURL.ValueString(),
		UIAddress:             data.UIAddress.ValueString(),
		UIFQDN:                data.UIFQDN.ValueString(),
	}

	// Encode the API data
	encodedBuffer := new(bytes.Buffer)
	if err := json.NewEncoder(encodedBuffer).Encode(apiData); err != nil {
		return errors.New("invalid format for tenant data")
	}

	tflog.Debug(ctx, fmt.Sprintf("Creating tenant with data: %v", encodedBuffer.String()))

	// Call the API
	apiResp, err := tc.client.Post(TenantEndpoint, encodedBuffer)
	if err != nil {
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	if apiResp.StatusCode != 201 {
		return fmt.Errorf("API returned status %d", apiResp.StatusCode)
	}

	// Decode the API response
	var tenantAPIResp vergeio.VergeResponse
	if err := json.NewDecoder(apiResp.Body).Decode(&tenantAPIResp); err != nil {
		return errors.New("invalid format received from API")
	}

	// Fill the data.id with the new tenant id from the API
	data.Id = types.StringValue(tenantAPIResp.Key)
	tflog.Debug(ctx, fmt.Sprintf("Created tenant with Id %v", data.Id.ValueString()))

	// If ui_address is set, update the vnet_address owner to enable auto-routing
	if !data.UIAddress.IsNull() && !data.UIAddress.IsUnknown() && data.UIAddress.ValueString() != "" {
		tflog.Debug(ctx, fmt.Sprintf("Setting vnet_address %s owner to tenants/%s", data.UIAddress.ValueString(), data.Id.ValueString()))
		
		ownerPayload := map[string]string{
			"owner": fmt.Sprintf("tenants/%s", data.Id.ValueString()),
		}
		ownerBuffer := new(bytes.Buffer)
		if err := json.NewEncoder(ownerBuffer).Encode(ownerPayload); err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Failed to encode owner payload: %v", err))
		} else {
			endpoint := fmt.Sprintf("api/v4/vnet_addresses/%s", data.UIAddress.ValueString())
			ownerResp, err := tc.client.Put(endpoint, ownerBuffer)
			if err != nil {
				tflog.Warn(ctx, fmt.Sprintf("Failed to set vnet_address owner: %v", err))
			} else if ownerResp.StatusCode >= 400 {
				tflog.Warn(ctx, fmt.Sprintf("Failed to set vnet_address owner: status %d", ownerResp.StatusCode))
			} else {
				tflog.Debug(ctx, "Successfully set vnet_address owner for auto-routing")
			}
		}
	}

	return nil
}

// Update the tenant in the API.
func (tc *TenantApi) updateTenant(ctx context.Context, planData *TenantResourceModel, stateData *TenantResourceModel) error {

	apiData := TenantAPIResourceModel{
		Name:                  vergeio.StringToNil(planData.Name, stateData.Name, ""),
		Description:           vergeio.StringToNil(planData.Description, stateData.Description, ""),
		ExposeCloudSnapshots:  planData.ExposeCloudSnapshots.ValueBool(),
		AllowBranding:         planData.AllowBranding.ValueBool(),
		URL:                   vergeio.StringToNil(planData.URL, stateData.URL, ""),
		Note:                  vergeio.StringToNil(planData.Note, stateData.Note, ""),
		Isolate:               planData.Isolate.ValueBool(),
		ThemeAccess:           vergeio.StringToNil(planData.ThemeAccess, stateData.ThemeAccess, ""),
		HelpURL:               vergeio.StringToNil(planData.HelpURL, stateData.HelpURL, ""),
		UIAddress:             vergeio.StringToNil(planData.UIAddress, stateData.UIAddress, ""),
		UIFQDN:                vergeio.StringToNil(planData.UIFQDN, stateData.UIFQDN, ""),
	}

	// Encode the API data
	encodedBuffer := new(bytes.Buffer)
	if err := json.NewEncoder(encodedBuffer).Encode(apiData); err != nil {
		return errors.New("invalid format for tenant data")
	}

	tflog.Debug(ctx, fmt.Sprintf("Updating tenant with data: %v", encodedBuffer.String()))

	// Call the API
	apiResp, err := tc.client.Put(fmt.Sprintf("%s/%s",
		TenantEndpoint,
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

	tflog.Debug(ctx, fmt.Sprintf("Updated tenant %v", stateData.Id.ValueString()))

	defer apiResp.Body.Close()

	return nil
}

// Read the tenant from the API.
func (tc *TenantApi) readTenant(ctx context.Context, data *TenantResourceModel) error {

	tflog.Debug(ctx, "Reading tenant data")

	// Call the Get API with the tenant id
	apiResp, err := tc.client.Get(fmt.Sprintf("%s/%s",
		TenantEndpoint,
		url.PathEscape(data.Id.ValueString()),
	), &vergeio.Options{Fields: "$key,name,description,expose_cloud_snapshots,allow_branding,url,note,isolate,theme_access,help_url,ui_address,ui_fqdn,uuid,created,modified,owner,creator,vnet"})

	if err != nil {
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	if apiResp.StatusCode == 404 {
		return errors.New("tenant not found")
	}
	if apiResp.StatusCode != 200 {
		return fmt.Errorf("API returned status %d", apiResp.StatusCode)
	}

	// Decode the API response
	var tenantAPIResp TenantAPIResponseModel
	if err := json.NewDecoder(apiResp.Body).Decode(&tenantAPIResp); err != nil {
		return errors.New("invalid format received from API")
	}

	// Map API response to resource model
	data.Name = types.StringValue(tenantAPIResp.Name)
	data.Description = types.StringValue(tenantAPIResp.Description)
	data.ExposeCloudSnapshots = types.BoolValue(tenantAPIResp.ExposeCloudSnapshots)
	data.AllowBranding = types.BoolValue(tenantAPIResp.AllowBranding)
	data.URL = types.StringValue(tenantAPIResp.URL)
	data.Note = types.StringValue(tenantAPIResp.Note)
	data.Isolate = types.BoolValue(tenantAPIResp.Isolate)
	data.ThemeAccess = types.StringValue(tenantAPIResp.ThemeAccess)
	data.HelpURL = types.StringValue(tenantAPIResp.HelpURL)
	
	// Handle nullable interface fields (can be string, int, or null)
	if tenantAPIResp.UIAddress != nil {
		switch v := tenantAPIResp.UIAddress.(type) {
		case string:
			data.UIAddress = types.StringValue(v)
		case float64:
			data.UIAddress = types.StringValue(fmt.Sprintf("%.0f", v))
		default:
			data.UIAddress = types.StringValue("")
		}
	} else {
		data.UIAddress = types.StringValue("")
	}
	if tenantAPIResp.UIFQDN != nil {
		switch v := tenantAPIResp.UIFQDN.(type) {
		case string:
			data.UIFQDN = types.StringValue(v)
		default:
			data.UIFQDN = types.StringValue("")
		}
	} else {
		data.UIFQDN = types.StringValue("")
	}
	data.UUID = types.StringValue(tenantAPIResp.UUID)
	data.VNetID = types.Int64Value(int64(tenantAPIResp.VNet))

	tflog.Debug(ctx, "Tenant data was successfully read")

	return nil
}

// Delete the tenant from the API.
func (tc *TenantApi) deleteTenant(ctx context.Context, data *TenantResourceModel) error {

	tflog.Debug(ctx, "Deleting tenant")

	_, err := tc.client.Delete(fmt.Sprintf("%s/%s",
		TenantEndpoint,
		url.PathEscape(data.Id.ValueString())))

	if err != nil {
		return errors.New("error deleting tenant: " + err.Error())
	}

	tflog.Debug(ctx, "Tenant was successfully deleted")

	return nil
}
