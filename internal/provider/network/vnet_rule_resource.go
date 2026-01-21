// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

package network

import (
	"context"
	"strings"

	"terraform-provider-vergeio/internal/provider/vergeio"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &VNetRuleResource{}
var _ resource.ResourceWithImportState = &VNetRuleResource{}

func NewVNetRuleResource() resource.Resource {
	return &VNetRuleResource{}
}

type VNetRuleResource struct {
	vnetRuleApi *VNetRuleApi
}

type VNetRuleResourceModel struct {
	Id               types.String `tfsdk:"id"`
	VNet             types.Int64  `tfsdk:"vnet"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	OrderID          types.Int64  `tfsdk:"orderid"`
	Protocol         types.String `tfsdk:"protocol"`
	Direction        types.String `tfsdk:"direction"`
	Interface        types.String `tfsdk:"interface"`
	Action           types.String `tfsdk:"action"`
	SourceIP         types.String `tfsdk:"source_ip"`
	SourcePorts      types.String `tfsdk:"source_ports"`
	DestinationIP    types.String `tfsdk:"destination_ip"`
	DestinationPorts types.String `tfsdk:"destination_ports"`
	TargetIP         types.String `tfsdk:"target_ip"`
	TargetPorts      types.String `tfsdk:"target_ports"`
}

func (r *VNetRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vnet_rule"
}

func (r *VNetRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a VergeOS Virtual Network Rule (Firewall Rule).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the rule.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vnet": schema.Int64Attribute{
				MarkdownDescription: "The ID of the virtual network this rule belongs to.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the rule.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the rule.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the rule is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"orderid": schema.Int64Attribute{
				MarkdownDescription: "Ordering/Priority of the rule.",
				Optional:            true,
				Computed:            true,
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "Protocol (tcp, udp, icmp, any, etc).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("any"),
				Validators: []validator.String{
					stringvalidator.OneOf("tcp", "tcpudp", "udp", "icmp", "89", "2", "47", "50", "51", "any"),
				},
			},
			"direction": schema.StringAttribute{
				MarkdownDescription: "Direction (incoming, outgoing).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("incoming"),
				Validators: []validator.String{
					stringvalidator.OneOf("incoming", "outgoing"),
				},
			},
			"interface": schema.StringAttribute{
				MarkdownDescription: "Interface (auto, router, dmz, wireguard, any).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("auto"),
				Validators: []validator.String{
					stringvalidator.OneOf("auto", "router", "dmz", "wireguard", "any"),
				},
			},
			"action": schema.StringAttribute{
				MarkdownDescription: "Action (accept, drop, reject, translate, route).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("accept"),
				Validators: []validator.String{
					stringvalidator.OneOf("accept", "drop", "reject", "translate", "route"),
				},
			},
			"source_ip": schema.StringAttribute{
				MarkdownDescription: "Source IP address or CIDR.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"source_ports": schema.StringAttribute{
				MarkdownDescription: "Source port(s).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"destination_ip": schema.StringAttribute{
				MarkdownDescription: "Destination IP address or CIDR.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"destination_ports": schema.StringAttribute{
				MarkdownDescription: "Destination port(s).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"target_ip": schema.StringAttribute{
				MarkdownDescription: "Target IP for translation (NAT).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"target_ports": schema.StringAttribute{
				MarkdownDescription: "Target port(s) for translation (NAT).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
		},
	}
}

func (r *VNetRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*vergeio.Client)
	if !ok {
		return
	}

	r.vnetRuleApi = NewVNetRuleApi(client)
}

func (r *VNetRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VNetRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.vnetRuleApi.createVNetRule(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error Creating VNet Rule", err.Error())
		return
	}

	if err := r.vnetRuleApi.readVNetRule(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error Reading VNet Rule", err.Error())
		return
	}

	// Apply rules to the VNet
	if err := r.vnetRuleApi.applyVNetRules(ctx, data.VNet.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Error Applying VNet Rules", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VNetRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VNetRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.vnetRuleApi.readVNetRule(ctx, &data); err != nil {
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading VNet Rule", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VNetRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData VNetRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.vnetRuleApi.updateVNetRule(ctx, &planData, &stateData); err != nil {
		resp.Diagnostics.AddError("Error Updating VNet Rule", err.Error())
		return
	}

	if err := r.vnetRuleApi.readVNetRule(ctx, &stateData); err != nil {
		resp.Diagnostics.AddError("Error Reading VNet Rule", err.Error())
		return
	}

	// Apply rules to the VNet
	if err := r.vnetRuleApi.applyVNetRules(ctx, stateData.VNet.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Error Applying VNet Rules", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
}

func (r *VNetRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VNetRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.vnetRuleApi.deleteVNetRule(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error Deleting VNet Rule", err.Error())
		return
	}

	// Apply rules to the VNet
	if err := r.vnetRuleApi.applyVNetRules(ctx, data.VNet.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Error Applying VNet Rules", err.Error())
		return
	}
}

func (r *VNetRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
