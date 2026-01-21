package vm

import (
	"context"
	"fmt"

	"terraform-provider-vergeio/internal/provider/vergeio"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &VMSnapshotResource{}
var _ resource.ResourceWithImportState = &VMSnapshotResource{}

func NewVMSnapshotResource() resource.Resource {
	return &VMSnapshotResource{}
}

// VMSnapshotResource defines the resource implementation.
type VMSnapshotResource struct {
	vmApi *VMApi
}

// VMSnapshotResourceModel describes the resource data model.
type VMSnapshotResourceModel struct {
	Id            types.String `tfsdk:"id"`
	VMId          types.String `tfsdk:"vm_id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	MachineId     types.Int32  `tfsdk:"machine_id"`
	SnapMachineId types.Int32  `tfsdk:"snap_machine_id"`
	Created       types.Int64  `tfsdk:"created"`
}

func (r *VMSnapshotResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_snapshot"
}

func (r *VMSnapshotResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a VM snapshot in VergeOS.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the snapshot.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vm_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the VM to snapshot.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the snapshot.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "A description of the snapshot.",
			},
			"machine_id": schema.Int32Attribute{
				Computed:            true,
				MarkdownDescription: "The ID of the parent machine.",
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			"snap_machine_id": schema.Int32Attribute{
				Computed:            true,
				MarkdownDescription: "The ID of the snapshot machine.",
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			"created": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the snapshot was created.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *VMSnapshotResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.vmApi = NewVMApi(client)
}

func (r *VMSnapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VMSnapshotResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.vmApi.CreateVMSnapshot(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating VM Snapshot", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VMSnapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VMSnapshotResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.vmApi.ReadVMSnapshot(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading VM Snapshot", err.Error())
		return
	}

	if data.Id.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VMSnapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VMSnapshotResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.vmApi.UpdateVMSnapshot(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating VM Snapshot", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VMSnapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VMSnapshotResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.vmApi.DeleteVMSnapshot(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting VM Snapshot", err.Error())
		return
	}
}

func (r *VMSnapshotResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
