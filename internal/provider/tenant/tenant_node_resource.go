// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

package tenant

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-vergeio/internal/provider/vergeio"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &TenantNodeResource{}
var _ resource.ResourceWithImportState = &TenantNodeResource{}

func NewTenantNodeResource() resource.Resource {
	return &TenantNodeResource{}
}

type TenantNodeResource struct {
	tenantNodeApi *TenantNodeApi
}

// TenantNodeResourceModel describes the resource data model.
type TenantNodeResourceModel struct {
	Id          types.String `tfsdk:"id"`
	TenantID    types.Int64  `tfsdk:"tenant_id"`
	NodeID      types.Int64  `tfsdk:"node_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Powerstate  types.Bool   `tfsdk:"powerstate"`
	CPUCores    types.Int64  `tfsdk:"cpu_cores"`
	RAM         types.Int64  `tfsdk:"ram"`
	OnPowerLoss types.String `tfsdk:"on_power_loss"`
}

func (r *TenantNodeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant_node"
}

func (r *TenantNodeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a VergeOS Tenant Node. Tenant nodes allocate CPU cores and RAM to a tenant, enabling it to run virtual workloads.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the tenant node.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tenant_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the tenant this node belongs to.",
				Required:            true,
			},
			"node_id": schema.Int64Attribute{
				MarkdownDescription: "The node ID within the tenant (1-65535). Typically starts at 1.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the tenant node.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("node1"),
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the tenant node.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the tenant node is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"powerstate": schema.BoolAttribute{
				MarkdownDescription: "Whether the tenant node should be powered on.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"cpu_cores": schema.Int64Attribute{
				MarkdownDescription: "Number of CPU cores to allocate to this tenant node. Minimum 1.",
				Required:            true,
			},
			"ram": schema.Int64Attribute{
				MarkdownDescription: "Amount of RAM in MB to allocate to this tenant node. Minimum 2048 MB (2 GB).",
				Required:            true,
			},
			"on_power_loss": schema.StringAttribute{
				MarkdownDescription: "Behavior on power loss. Valid values: `power_on`, `last_state`, `leave_off`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("last_state"),
				Validators: []validator.String{
					stringvalidator.OneOf("power_on", "last_state", "leave_off"),
				},
			},
		},
	}
}

func (r *TenantNodeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*vergeio.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *vergeio.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.tenantNodeApi = NewTenantNodeApi(client)
}

func (r *TenantNodeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TenantNodeResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.tenantNodeApi.createTenantNode(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error Creating Tenant Node", err.Error())
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Created tenant node with id %v", data.Id.ValueString()))

	if err := r.tenantNodeApi.readTenantNode(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error Reading Tenant Node", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TenantNodeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TenantNodeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.tenantNodeApi.readTenantNode(ctx, &data); err != nil {
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Tenant Node", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TenantNodeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData TenantNodeResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.tenantNodeApi.updateTenantNode(ctx, &planData, &stateData); err != nil {
		resp.Diagnostics.AddError("Error Updating Tenant Node", err.Error())
		return
	}

	if err := r.tenantNodeApi.readTenantNode(ctx, &stateData); err != nil {
		resp.Diagnostics.AddError("Error Reading Tenant Node", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
}

func (r *TenantNodeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TenantNodeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.tenantNodeApi.deleteTenantNode(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error Deleting Tenant Node", err.Error())
		return
	}

	tflog.Debug(ctx, "Tenant node was successfully deleted")
}

func (r *TenantNodeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
