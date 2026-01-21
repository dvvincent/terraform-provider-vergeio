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
	VNetRuleEndpoint = vergeio.APIEndpoint + "/vnet_rules"
)

func NewVNetRuleApi(c *vergeio.Client) *VNetRuleApi {
	return &VNetRuleApi{
		name:   "VNet Rule Api",
		client: c,
	}
}

type VNetRuleApi struct {
	name   string
	client *vergeio.Client
}

func (va *VNetRuleApi) Name() string {
	return va.name
}

type VNetRuleAPIModel struct {
	VNet             int    `json:"vnet,omitempty"`
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	Enabled          bool   `json:"enabled"`
	OrderID          int    `json:"orderid,omitempty"`
	Protocol         string `json:"protocol,omitempty"`
	Direction        string `json:"direction,omitempty"`
	Interface        string `json:"interface,omitempty"`
	Action           string `json:"action,omitempty"`
	SourceIP         string `json:"source_ip,omitempty"`
	SourcePorts      string `json:"source_ports,omitempty"`
	DestinationIP    string `json:"destination_ip,omitempty"`
	DestinationPorts string `json:"destination_ports,omitempty"`
	TargetIP         string `json:"target_ip,omitempty"`
	TargetPorts      string `json:"target_ports,omitempty"`
}

type VNetRuleAPIResponseModel struct {
	Key              int    `json:"$key,omitempty"`
	VNet             int    `json:"vnet,omitempty"`
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	Enabled          bool   `json:"enabled"`
	OrderID          int    `json:"orderid,omitempty"`
	Protocol         string `json:"protocol,omitempty"`
	Direction        string `json:"direction,omitempty"`
	Interface        string `json:"interface,omitempty"`
	Action           string `json:"action,omitempty"`
	SourceIP         string `json:"source_ip,omitempty"`
	SourcePorts      string `json:"source_ports,omitempty"`
	DestinationIP    string `json:"destination_ip,omitempty"`
	DestinationPorts string `json:"destination_ports,omitempty"`
	TargetIP         string `json:"target_ip,omitempty"`
	TargetPorts      string `json:"target_ports,omitempty"`
}

func (va *VNetRuleApi) createVNetRule(ctx context.Context, data *VNetRuleResourceModel) error {
	vnetID := int(data.VNet.ValueInt64())

	apiData := VNetRuleAPIModel{
		VNet:             vnetID,
		Name:             data.Name.ValueString(),
		Description:      data.Description.ValueString(),
		Enabled:          data.Enabled.ValueBool(),
		OrderID:          int(data.OrderID.ValueInt64()),
		Protocol:         data.Protocol.ValueString(),
		Direction:        data.Direction.ValueString(),
		Interface:        data.Interface.ValueString(),
		Action:           data.Action.ValueString(),
		SourceIP:         data.SourceIP.ValueString(),
		SourcePorts:      data.SourcePorts.ValueString(),
		DestinationIP:    data.DestinationIP.ValueString(),
		DestinationPorts: data.DestinationPorts.ValueString(),
		TargetIP:         data.TargetIP.ValueString(),
		TargetPorts:      data.TargetPorts.ValueString(),
	}

	encodedBuffer := new(bytes.Buffer)
	if err := json.NewEncoder(encodedBuffer).Encode(apiData); err != nil {
		return errors.New("invalid format for vnet rule data")
	}

	apiResp, err := va.client.Post(VNetRuleEndpoint, encodedBuffer)
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
	return nil
}

func (va *VNetRuleApi) updateVNetRule(ctx context.Context, planData *VNetRuleResourceModel, stateData *VNetRuleResourceModel) error {
	apiData := VNetRuleAPIModel{
		Name:             planData.Name.ValueString(),
		Description:      planData.Description.ValueString(),
		Enabled:          planData.Enabled.ValueBool(),
		OrderID:          int(planData.OrderID.ValueInt64()),
		Protocol:         planData.Protocol.ValueString(),
		Direction:        planData.Direction.ValueString(),
		Interface:        planData.Interface.ValueString(),
		Action:           planData.Action.ValueString(),
		SourceIP:         planData.SourceIP.ValueString(),
		SourcePorts:      planData.SourcePorts.ValueString(),
		DestinationIP:    planData.DestinationIP.ValueString(),
		DestinationPorts: planData.DestinationPorts.ValueString(),
		TargetIP:         planData.TargetIP.ValueString(),
		TargetPorts:      planData.TargetPorts.ValueString(),
	}

	encodedBuffer := new(bytes.Buffer)
	if err := json.NewEncoder(encodedBuffer).Encode(apiData); err != nil {
		return errors.New("invalid format for vnet rule data")
	}

	apiResp, err := va.client.Put(fmt.Sprintf("%s/%s",
		VNetRuleEndpoint,
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

func (va *VNetRuleApi) readVNetRule(ctx context.Context, data *VNetRuleResourceModel) error {
	apiResp, err := va.client.Get(fmt.Sprintf("%s/%s",
		VNetRuleEndpoint,
		url.PathEscape(data.Id.ValueString()),
	), &vergeio.Options{Fields: "$key,vnet,name,description,enabled,orderid,protocol,direction,interface,action,source_ip,source_ports,destination_ip,destination_ports,target_ip,target_ports"})

	if err != nil {
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	if apiResp.StatusCode == 404 {
		return errors.New("vnet rule not found")
	}
	if apiResp.StatusCode != 200 {
		return fmt.Errorf("API returned status %d", apiResp.StatusCode)
	}

	var apiModel VNetRuleAPIResponseModel
	if err := json.NewDecoder(apiResp.Body).Decode(&apiModel); err != nil {
		return errors.New("invalid format received from API")
	}

	data.VNet = types.Int64Value(int64(apiModel.VNet))
	data.Name = types.StringValue(apiModel.Name)
	data.Description = types.StringValue(apiModel.Description)
	data.Enabled = types.BoolValue(apiModel.Enabled)
	data.OrderID = types.Int64Value(int64(apiModel.OrderID))
	data.Protocol = types.StringValue(apiModel.Protocol)
	data.Direction = types.StringValue(apiModel.Direction)
	data.Interface = types.StringValue(apiModel.Interface)
	data.Action = types.StringValue(apiModel.Action)
	data.SourceIP = types.StringValue(apiModel.SourceIP)
	data.SourcePorts = types.StringValue(apiModel.SourcePorts)
	data.DestinationIP = types.StringValue(apiModel.DestinationIP)
	data.DestinationPorts = types.StringValue(apiModel.DestinationPorts)
	data.TargetIP = types.StringValue(apiModel.TargetIP)
	data.TargetPorts = types.StringValue(apiModel.TargetPorts)

	return nil
}

func (va *VNetRuleApi) deleteVNetRule(ctx context.Context, data *VNetRuleResourceModel) error {
	_, err := va.client.Delete(fmt.Sprintf("%s/%s",
		VNetRuleEndpoint,
		url.PathEscape(data.Id.ValueString())))

	if err != nil {
		return errors.New("error deleting vnet rule: " + err.Error())
	}

	return nil
}
func (va *VNetRuleApi) applyVNetRules(ctx context.Context, vnetID int64) error {
	actionPayload := VNetAction{
		VNet:   int(vnetID),
		Action: "refresh",
		Params: json.RawMessage("{}"),
	}

	bytedata, err := json.Marshal(actionPayload)
	if err != nil {
		return err
	}

	apiResp, err := va.client.Post(NetworkActionEndpoint, bytes.NewBuffer(bytedata))
	if err != nil {
		return err
	}
	if apiResp.StatusCode != 201 {
		return fmt.Errorf("failed to apply vnet rules: API returned status %d", apiResp.StatusCode)
	}

	return nil
}
