// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

package network

import (
	"context"
	"fmt"

	"terraform-provider-vergeio/internal/provider/vergeio"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &VNetAddressResource{}

func NewVNetAddressResource() resource.Resource {
	return &VNetAddressResource{}
}

// VNetAddressResource defines the resource implementation.
type VNetAddressResource struct {
	client *vergeio.Client
}

// VNetAddressResourceModel describes the resource data model.
type VNetAddressResourceModel struct {
	Id          types.String `tfsdk:"id"`
	VNet        types.Int64  `tfsdk:"vnet"`
	IP          types.String `tfsdk:"ip"`
	Type        types.String `tfsdk:"type"`
	Hostname    types.String `tfsdk:"hostname"`
	Description types.String `tfsdk:"description"`
	Owner       types.String `tfsdk:"owner"`
	MAC         types.String `tfsdk:"mac"`
}

func (r *VNetAddressResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vnet_address"
}

func (r *VNetAddressResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a virtual network IP address. Can be used to allocate public IPs for tenant UI access.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the VNet address",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vnet": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "The ID of the virtual network to allocate the address on",
			},
			"ip": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The IP address. If not specified, one will be auto-assigned from the network's pool",
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Address type: 'virtual', 'static', 'dynamic', 'ipalias', or 'proxy'",
			},
			"hostname": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Hostname for DNS registration",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Description of the address",
			},
			"owner": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The owner of the address (e.g., 'tenants/1')",
			},
			"mac": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "MAC address if associated with a NIC",
			},
		},
	}
}

func (r *VNetAddressResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*vergeio.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *vergeio.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *VNetAddressResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VNetAddressResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vnetAddressApi := NewVNetAddressApi(r.client)
	if err := vnetAddressApi.createVNetAddress(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error Creating VNet Address", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VNetAddressResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VNetAddressResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vnetAddressApi := NewVNetAddressApi(r.client)
	if err := vnetAddressApi.readVNetAddress(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error Reading VNet Address", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VNetAddressResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData VNetAddressResourceModel
	var stateData VNetAddressResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vnetAddressApi := NewVNetAddressApi(r.client)
	if err := vnetAddressApi.updateVNetAddress(ctx, &planData, &stateData); err != nil {
		resp.Diagnostics.AddError("Error Updating VNet Address", err.Error())
		return
	}

	// Read back to get computed fields
	planData.Id = stateData.Id
	if err := vnetAddressApi.readVNetAddress(ctx, &planData); err != nil {
		resp.Diagnostics.AddError("Error Reading VNet Address after Update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (r *VNetAddressResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VNetAddressResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vnetAddressApi := NewVNetAddressApi(r.client)
	if err := vnetAddressApi.deleteVNetAddress(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error Deleting VNet Address", err.Error())
		return
	}
}
