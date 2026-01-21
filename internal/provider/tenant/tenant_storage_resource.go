// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

package tenant

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-vergeio/internal/provider/vergeio"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &TenantStorageResource{}
var _ resource.ResourceWithImportState = &TenantStorageResource{}

func NewTenantStorageResource() resource.Resource {
	return &TenantStorageResource{}
}

type TenantStorageResource struct {
	tenantStorageApi *TenantStorageApi
}

// TenantStorageResourceModel describes the resource data model.
type TenantStorageResourceModel struct {
	Id          types.String `tfsdk:"id"`
	TenantID    types.Int64  `tfsdk:"tenant_id"`
	Tier        types.Int64  `tfsdk:"tier"`
	Provisioned types.Int64  `tfsdk:"provisioned"`
	Used        types.Int64  `tfsdk:"used"`
	Allocated   types.Int64  `tfsdk:"allocated"`
}

func (r *TenantStorageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant_storage"
}

func (r *TenantStorageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a VergeOS Tenant Storage allocation. This resource provisions storage from a tier to a tenant.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the tenant storage allocation.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tenant_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the tenant to allocate storage for.",
				Required:            true,
			},
			"tier": schema.Int64Attribute{
				MarkdownDescription: "The storage tier to allocate from (0-5). Typically tier 1 is the default.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
			},
			"provisioned": schema.Int64Attribute{
				MarkdownDescription: "Amount of storage to provision in bytes. This is the maximum storage the tenant can use.",
				Required:            true,
			},
			"used": schema.Int64Attribute{
				MarkdownDescription: "Amount of storage currently used in bytes. (Computed)",
				Computed:            true,
			},
			"allocated": schema.Int64Attribute{
				MarkdownDescription: "Amount of storage currently allocated in bytes. (Computed)",
				Computed:            true,
			},
		},
	}
}

func (r *TenantStorageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.tenantStorageApi = NewTenantStorageApi(client)
}

func (r *TenantStorageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TenantStorageResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.tenantStorageApi.createTenantStorage(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error Creating Tenant Storage", err.Error())
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Created tenant storage with id %v", data.Id.ValueString()))

	if err := r.tenantStorageApi.readTenantStorage(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error Reading Tenant Storage", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TenantStorageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TenantStorageResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.tenantStorageApi.readTenantStorage(ctx, &data); err != nil {
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Tenant Storage", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TenantStorageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData TenantStorageResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.tenantStorageApi.updateTenantStorage(ctx, &planData, &stateData); err != nil {
		resp.Diagnostics.AddError("Error Updating Tenant Storage", err.Error())
		return
	}

	if err := r.tenantStorageApi.readTenantStorage(ctx, &stateData); err != nil {
		resp.Diagnostics.AddError("Error Reading Tenant Storage", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
}

func (r *TenantStorageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TenantStorageResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.tenantStorageApi.deleteTenantStorage(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error Deleting Tenant Storage", err.Error())
		return
	}

	tflog.Debug(ctx, "Tenant storage was successfully deleted")
}

func (r *TenantStorageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
