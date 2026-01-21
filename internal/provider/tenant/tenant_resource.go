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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TenantResource{}
var _ resource.ResourceWithImportState = &TenantResource{}

func NewTenantResource() resource.Resource {
	return &TenantResource{}
}

// TenantResource defines the resource implementation.
type TenantResource struct {
	tenantApi *TenantApi
}

// TenantResourceModel describes the resource data model.
type TenantResourceModel struct {
	Id                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Password             types.String `tfsdk:"password"`
	ChangePassword       types.Bool   `tfsdk:"change_password"`
	ExposeCloudSnapshots types.Bool   `tfsdk:"expose_cloud_snapshots"`
	AllowBranding        types.Bool   `tfsdk:"allow_branding"`
	URL                  types.String `tfsdk:"url"`
	Note                 types.String `tfsdk:"note"`
	Isolate              types.Bool   `tfsdk:"isolate"`
	ThemeAccess          types.String `tfsdk:"theme_access"`
	HelpURL              types.String `tfsdk:"help_url"`
	UIAddress            types.String `tfsdk:"ui_address"`
	UIFQDN               types.String `tfsdk:"ui_fqdn"`
	UUID                 types.String `tfsdk:"uuid"`
	VNetID               types.Int64  `tfsdk:"vnet_id"`
}

// Metadata returns the resource type name.
func (r *TenantResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant"
}

func (r *TenantResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a VergeOS Tenant. Tenants provide multi-tenancy capabilities, allowing you to create isolated environments within a VergeOS system.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the tenant.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the tenant. Must be unique.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the tenant.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The admin password for the tenant. This is only used during creation.",
				Optional:            true,
				Sensitive:           true,
			},
			"change_password": schema.BoolAttribute{
				MarkdownDescription: "Whether to force a password change on first login.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"expose_cloud_snapshots": schema.BoolAttribute{
				MarkdownDescription: "Whether to expose cloud snapshots to the tenant.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"allow_branding": schema.BoolAttribute{
				MarkdownDescription: "Whether to allow the tenant to customize branding.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL associated with the tenant.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"note": schema.StringAttribute{
				MarkdownDescription: "Administrative notes for the tenant.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"isolate": schema.BoolAttribute{
				MarkdownDescription: "Whether the tenant is in isolation mode.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"theme_access": schema.StringAttribute{
				MarkdownDescription: "Theme access level. Valid values: `specified`, `host_only`, `local_only`, `both`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("host_only"),
				Validators: []validator.String{
					stringvalidator.OneOf("specified", "host_only", "local_only", "both"),
				},
			},
			"help_url": schema.StringAttribute{
				MarkdownDescription: "Custom help URL for the tenant.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("default"),
			},
			"ui_address": schema.StringAttribute{
				MarkdownDescription: "The UI address (IP) for the tenant. Can be an ID of a VNet Address.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"ui_fqdn": schema.StringAttribute{
				MarkdownDescription: "The fully qualified domain name for the tenant UI.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the tenant. (Computed)",
				Computed:            true,
			},
			"vnet_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the default virtual network created for this tenant. (Computed)",
				Computed:            true,
			},
		},
	}
}

func (r *TenantResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.tenantApi = NewTenantApi(client)
}

// Create a new tenant.
func (r *TenantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TenantResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.tenantApi.createTenant(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error Creating Tenant", err.Error())
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Created tenant with id %v", data.Id.ValueString()))

	// Read the tenant to get all computed attributes
	if err := r.tenantApi.readTenant(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error Reading Tenant", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read a tenant.
func (r *TenantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TenantResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.tenantApi.readTenant(ctx, &data); err != nil {
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Error Reading Tenant", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update a tenant.
func (r *TenantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData TenantResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.tenantApi.updateTenant(ctx, &planData, &stateData); err != nil {
		resp.Diagnostics.AddError("Error Updating Tenant", err.Error())
		return
	}

	// Read the tenant to get updated computed attributes
	if err := r.tenantApi.readTenant(ctx, &stateData); err != nil {
		resp.Diagnostics.AddError("Error Reading Tenant", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
}

// Delete a tenant.
func (r *TenantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TenantResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.tenantApi.deleteTenant(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error Deleting Tenant", err.Error())
		return
	}

	tflog.Debug(ctx, "Tenant was successfully deleted")
}

func (r *TenantResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
