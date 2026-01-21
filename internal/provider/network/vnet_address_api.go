// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"terraform-provider-vergeio/internal/provider/vergeio"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	VNetAddressEndpoint = vergeio.APIEndpoint + "/vnet_addresses"
)

func NewVNetAddressApi(c *vergeio.Client) *VNetAddressApi {
	return &VNetAddressApi{
		name:   "VNet Address Api",
		client: c,
	}
}

type VNetAddressApi struct {
	name   string
	client *vergeio.Client
}

func (va *VNetAddressApi) Name() string {
	return va.name
}

type VNetAddressAPIModel struct {
	VNet        int    `json:"vnet"`
	IP          string `json:"ip,omitempty"`
	Type        string `json:"type"`
	Hostname    string `json:"hostname,omitempty"`
	Description string `json:"description,omitempty"`
	Owner       string `json:"owner,omitempty"`
}

type VNetAddressAPIResponseModel struct {
	Key         int    `json:"$key,omitempty"`
	VNet        int    `json:"vnet,omitempty"`
	IP          string `json:"ip,omitempty"`
	Type        string `json:"type,omitempty"`
	Hostname    string `json:"hostname,omitempty"`
	Description string `json:"description,omitempty"`
	Owner       string `json:"owner,omitempty"`
	MAC         string `json:"mac,omitempty"`
}

func (va *VNetAddressApi) createVNetAddress(ctx context.Context, data *VNetAddressResourceModel) error {
	vnetID := int(data.VNet.ValueInt64())

	apiData := VNetAddressAPIModel{
		VNet:        vnetID,
		IP:          data.IP.ValueString(),
		Type:        data.Type.ValueString(),
		Hostname:    data.Hostname.ValueString(),
		Description: data.Description.ValueString(),
	}

	encodedBuffer := new(bytes.Buffer)
	if err := json.NewEncoder(encodedBuffer).Encode(apiData); err != nil {
		return errors.New("invalid format for vnet address data")
	}

	apiResp, err := va.client.Post(VNetAddressEndpoint, encodedBuffer)
	if err != nil {
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	if apiResp.StatusCode != 201 {
		return fmt.Errorf("API returned status %d", apiResp.StatusCode)
	}

	var vergeResp vergeio.VergeResponse
	if err := json.NewDecoder(apiResp.Body).Decode(&vergeResp); err != nil {
		return errors.New("invalid format received from API")
	}

	data.Id = types.StringValue(vergeResp.Key)

	// Read back to get the assigned IP if it was auto-assigned
	return va.readVNetAddress(ctx, data)
}

func (va *VNetAddressApi) updateVNetAddress(ctx context.Context, planData *VNetAddressResourceModel, stateData *VNetAddressResourceModel) error {
	apiData := VNetAddressAPIModel{
		IP:          planData.IP.ValueString(),
		Hostname:    planData.Hostname.ValueString(),
		Description: planData.Description.ValueString(),
	}

	encodedBuffer := new(bytes.Buffer)
	if err := json.NewEncoder(encodedBuffer).Encode(apiData); err != nil {
		return errors.New("invalid format for vnet address data")
	}

	apiResp, err := va.client.Put(fmt.Sprintf("%s/%s",
		VNetAddressEndpoint,
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

func (va *VNetAddressApi) readVNetAddress(ctx context.Context, data *VNetAddressResourceModel) error {
	apiResp, err := va.client.Get(fmt.Sprintf("%s/%s",
		VNetAddressEndpoint,
		url.PathEscape(data.Id.ValueString()),
	), &vergeio.Options{Fields: "$key,vnet,ip,type,hostname,description,owner,mac"})

	if err != nil {
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	if apiResp.StatusCode == 404 {
		return errors.New("vnet address not found")
	}
	if apiResp.StatusCode != 200 {
		return fmt.Errorf("API returned status %d", apiResp.StatusCode)
	}

	var apiModel VNetAddressAPIResponseModel
	if err := json.NewDecoder(apiResp.Body).Decode(&apiModel); err != nil {
		return errors.New("invalid format received from API")
	}

	data.VNet = types.Int64Value(int64(apiModel.VNet))
	data.IP = types.StringValue(apiModel.IP)
	data.Type = types.StringValue(apiModel.Type)
	data.Hostname = types.StringValue(apiModel.Hostname)
	data.Description = types.StringValue(apiModel.Description)
	data.Owner = types.StringValue(apiModel.Owner)
	data.MAC = types.StringValue(apiModel.MAC)

	return nil
}

func (va *VNetAddressApi) deleteVNetAddress(ctx context.Context, data *VNetAddressResourceModel) error {
	_, err := va.client.Delete(fmt.Sprintf("%s/%s",
		VNetAddressEndpoint,
		url.PathEscape(data.Id.ValueString())))

	if err != nil {
		return errors.New("error deleting vnet address: " + err.Error())
	}

	return nil
}
