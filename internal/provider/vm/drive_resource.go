package vm

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
var _ resource.Resource = &DriveResource{}

func NewDriveResource() resource.Resource {
return &DriveResource{}
}

// DriveResource defines the resource implementation.
type DriveResource struct {
diskApi *DiskApi
}

// DriveResourceModel describes the resource data model.
type DriveResourceModel struct {
Id                  types.String `tfsdk:"id"`
Machine             types.Int32  `tfsdk:"machine"`
Name                types.String `tfsdk:"name"`
Description         types.String `tfsdk:"description"`
Interface           types.String `tfsdk:"interface"`
Media               types.String `tfsdk:"media"`
MediaSource         types.Int32  `tfsdk:"media_source"`
DiskSize            types.Int64  `tfsdk:"disksize"`
PreferredTier       types.String `tfsdk:"preferred_tier"`
Enabled             types.Bool   `tfsdk:"enabled"`
ReadOnly            types.Bool   `tfsdk:"readonly"`
Serial              types.String `tfsdk:"serial"`
Asset               types.String `tfsdk:"asset"`
OrderId             types.Int32  `tfsdk:"orderid"`
PreserveDriveFormat types.Bool   `tfsdk:"preserve_drive_format"`
}

func (r *DriveResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
resp.TypeName = req.ProviderTypeName + "_drive"
}

func (r *DriveResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
resp.Schema = schema.Schema{
MarkdownDescription: "Standalone Drive resource in VergeIO. Can be attached to any machine by ID.",
Attributes: map[string]schema.Attribute{
"id": schema.StringAttribute{
Computed:            true,
MarkdownDescription: "The key of the drive.",
PlanModifiers: []planmodifier.String{
stringplanmodifier.UseStateForUnknown(),
},
},
"machine": schema.Int32Attribute{
Required:            true,
MarkdownDescription: "The machine ID to attach this drive to.",
},
"name": schema.StringAttribute{
Required:            true,
MarkdownDescription: "The name of the drive.",
},
"description": schema.StringAttribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "Description of the drive.",
},
"interface": schema.StringAttribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "Disk interface (e.g. virtio-scsi).",
},
"media": schema.StringAttribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "Media type (e.g. disk, cdrom, import).",
},
"media_source": schema.Int32Attribute{
Optional:            true,
MarkdownDescription: "Source file ID for the media (required for import).",
},
"disksize": schema.Int64Attribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "Disk size in GB.",
},
"preferred_tier": schema.StringAttribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "Storage tier (1-5).",
},
"enabled": schema.BoolAttribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "Whether the drive is enabled.",
},
"readonly": schema.BoolAttribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "Whether the drive is read-only.",
},
"serial": schema.StringAttribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "Serial number of the disk.",
},
"asset": schema.StringAttribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "Asset identification.",
},
"orderid": schema.Int32Attribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "Boot order or attachment order.",
},
"preserve_drive_format": schema.BoolAttribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "Whether to preserve drive format.",
},
},
}
}

func (r *DriveResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
if req.ProviderData == nil {
return
}
client, ok := req.ProviderData.(*vergeio.Client)
if !ok {
resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *vergeio.Client, got: %T", req.ProviderData))
return
}
r.diskApi = NewDiskApi(client)
}

func (r *DriveResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
var data DriveResourceModel
resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
if resp.Diagnostics.HasError() {
return
}

diskData := &diskResourceModel{
Machine:             data.Machine,
Name:                data.Name,
Description:         data.Description,
Interface:           data.Interface,
Media:               data.Media,
MediaSource:         data.MediaSource,
DiskSize:            data.DiskSize,
PreferredTier:       data.PreferredTier,
Enabled:             data.Enabled,
ReadOnly:            data.ReadOnly,
Serial:              data.Serial,
Asset:               data.Asset,
OrderId:             data.OrderId,
PreserveDriveFormat: data.PreserveDriveFormat,
}

if err := r.diskApi.createDisk(ctx, diskData); err != nil {
resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to create drive: %s", err))
return
}

data.Id = diskData.Key
data.Name = diskData.Name
data.Description = diskData.Description
data.Interface = diskData.Interface
data.Media = diskData.Media
data.MediaSource = diskData.MediaSource
data.DiskSize = diskData.DiskSize
data.PreferredTier = diskData.PreferredTier
data.Enabled = diskData.Enabled
data.ReadOnly = diskData.ReadOnly
data.Serial = diskData.Serial
data.Asset = diskData.Asset
data.OrderId = diskData.OrderId
data.PreserveDriveFormat = diskData.PreserveDriveFormat

resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DriveResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
var data DriveResourceModel
resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
if resp.Diagnostics.HasError() {
return
}

diskData := &diskResourceModel{
Key: data.Id,
}

if err := r.diskApi.readDisk(ctx, diskData); err != nil {
resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to read drive: %s", err))
return
}

data.Machine = diskData.Machine
data.Name = diskData.Name
data.Description = diskData.Description
data.Interface = diskData.Interface
data.Media = diskData.Media
data.MediaSource = diskData.MediaSource
data.DiskSize = diskData.DiskSize
data.PreferredTier = diskData.PreferredTier
data.Enabled = diskData.Enabled
data.ReadOnly = diskData.ReadOnly
data.Serial = diskData.Serial
data.Asset = diskData.Asset
data.OrderId = diskData.OrderId
data.PreserveDriveFormat = diskData.PreserveDriveFormat

resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DriveResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
var plan, state DriveResourceModel
resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
if resp.Diagnostics.HasError() {
return
}

planDiskData := &diskResourceModel{
Key:                 state.Id,
Machine:             plan.Machine,
Name:                plan.Name,
Description:         plan.Description,
Interface:           plan.Interface,
Media:               plan.Media,
MediaSource:         plan.MediaSource,
DiskSize:            plan.DiskSize,
PreferredTier:       plan.PreferredTier,
Enabled:             plan.Enabled,
ReadOnly:            plan.ReadOnly,
Serial:              plan.Serial,
Asset:               plan.Asset,
OrderId:             plan.OrderId,
PreserveDriveFormat: plan.PreserveDriveFormat,
}

stateDiskData := &diskResourceModel{
Key: state.Id,
}

if err := r.diskApi.updateDisk(ctx, planDiskData, stateDiskData); err != nil {
resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to update drive: %s", err))
return
}

plan.Id = stateDiskData.Key
plan.DiskSize = stateDiskData.DiskSize

resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DriveResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
var data DriveResourceModel
resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
if resp.Diagnostics.HasError() {
return
}

diskData := &diskResourceModel{
Key: data.Id,
}

// For standalone deletion, we use machine ID from state
vmId := types.StringValue(fmt.Sprintf("%d", data.Machine.ValueInt32()))

if err := r.diskApi.deleteDisk(ctx, diskData, vmId); err != nil {
resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to delete drive: %s", err))
return
}
}
