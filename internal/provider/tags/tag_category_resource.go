// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

package tags

import (
	"context"
	"fmt"

	"terraform-provider-vergeio/internal/provider/vergeio"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TagCategoryResource{}
var _ resource.ResourceWithImportState = &TagCategoryResource{}

func NewTagCategoryResource() resource.Resource {
	return &TagCategoryResource{}
}

// TagCategoryResource defines the resource implementation.
type TagCategoryResource struct {
	tagsApi *TagsApi
}

// TagCategoryResourceModel describes the resource data model.
type TagCategoryResourceModel struct {
	Id                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	Description             types.String `tfsdk:"description"`
	SingleTagSelection      types.Bool   `tfsdk:"single_tag_selection"`
	TaggableVMs             types.Bool   `tfsdk:"taggable_vms"`
	TaggableVolumes         types.Bool   `tfsdk:"taggable_volumes"`
	TaggableVnets           types.Bool   `tfsdk:"taggable_vnets"`
	TaggableVnetRules       types.Bool   `tfsdk:"taggable_vnet_rules"`
	TaggableTenants         types.Bool   `tfsdk:"taggable_tenants"`
	TaggableTenantNodes     types.Bool   `tfsdk:"taggable_tenant_nodes"`
	TaggableUsers           types.Bool   `tfsdk:"taggable_users"`
	TaggableNodes           types.Bool   `tfsdk:"taggable_nodes"`
	TaggableClusters        types.Bool   `tfsdk:"taggable_clusters"`
	TaggableGroups          types.Bool   `tfsdk:"taggable_groups"`
	TaggableSites           types.Bool   `tfsdk:"taggable_sites"`
	TaggableVmwareContainers types.Bool  `tfsdk:"taggable_vmware_containers"`
}

func (r *TagCategoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_category"
}

func (r *TagCategoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a tag category in VergeOS. Tag categories are containers that organize tags and define which resource types can be tagged.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the tag category.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the tag category.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "A description of the tag category.",
			},
			"single_tag_selection": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "If true, only one tag from this category can be assigned to a resource at a time.",
			},
			"taggable_vms": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Allow tagging VMs with tags from this category.",
			},
			"taggable_volumes": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Allow tagging Volumes with tags from this category.",
			},
			"taggable_vnets": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Allow tagging Virtual Networks with tags from this category.",
			},
			"taggable_vnet_rules": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Allow tagging VNet Rules with tags from this category.",
			},
			"taggable_tenants": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Allow tagging Tenants with tags from this category.",
			},
			"taggable_tenant_nodes": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Allow tagging Tenant Nodes with tags from this category.",
			},
			"taggable_users": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Allow tagging Users with tags from this category.",
			},
			"taggable_nodes": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Allow tagging Nodes with tags from this category.",
			},
			"taggable_clusters": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Allow tagging Clusters with tags from this category.",
			},
			"taggable_groups": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Allow tagging Groups with tags from this category.",
			},
			"taggable_sites": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Allow tagging Sites with tags from this category.",
			},
			"taggable_vmware_containers": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Allow tagging VMware Containers with tags from this category.",
			},
		},
	}
}

func (r *TagCategoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.tagsApi = NewTagsApi(client)
}

func (r *TagCategoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TagCategoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.tagsApi.createTagCategory(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Tag Category", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TagCategoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TagCategoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.tagsApi.readTagCategory(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Tag Category", err.Error())
		return
	}

	if data.Id.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TagCategoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data TagCategoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the current state to preserve the ID
	var state TagCategoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Id = state.Id

	err := r.tagsApi.updateTagCategory(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Tag Category", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TagCategoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TagCategoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.tagsApi.deleteTagCategory(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Tag Category", err.Error())
		return
	}
}

func (r *TagCategoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
