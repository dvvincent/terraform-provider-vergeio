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
var _ resource.Resource = &NICResource{}

func NewNICResource() resource.Resource {
return &NICResource{}
}

// NICResource defines the resource implementation.
type NICResource struct {
nicApi *NICApi
}

// NICResourceModel describes the resource data model.
type StandaloneNICResourceModel struct {
Id              types.String `tfsdk:"id"`
Machine         types.Int32  `tfsdk:"machine"`
Name            types.String `tfsdk:"name"`
Description     types.String `tfsdk:"description"`
Interface       types.String `tfsdk:"interface"`
Driver          types.String `tfsdk:"driver"`
Model           types.String `tfsdk:"model"`
Vendor          types.String `tfsdk:"vendor"`
Port            types.Int32  `tfsdk:"port"`
Enabled         types.Bool   `tfsdk:"enabled"`
VNET            types.Int32  `tfsdk:"vnet"`
MAC             types.String `tfsdk:"macaddress"`
IPAddress       types.String `tfsdk:"ipaddress"`
AssignIPAddress types.Bool   `tfsdk:"assign_ipaddress"`
Asset           types.String `tfsdk:"asset"`
AutoCreateVNet  types.Bool   `tfsdk:"auto_create_vnet"`
VNetUplink      types.Int32  `tfsdk:"vnet_uplink"`
}

func (r *NICResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
resp.TypeName = req.ProviderTypeName + "_nic"
}

func (r *NICResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
resp.Schema = schema.Schema{
MarkdownDescription: "Standalone NIC resource in VergeIO. Can be attached to any machine by ID.",
Attributes: map[string]schema.Attribute{
"id": schema.StringAttribute{
Computed:            true,
MarkdownDescription: "The key of the NIC.",
PlanModifiers: []planmodifier.String{
stringplanmodifier.UseStateForUnknown(),
},
},
"machine": schema.Int32Attribute{
Required:            true,
MarkdownDescription: "The machine ID to attach this NIC to.",
},
"name": schema.StringAttribute{
Required:            true,
MarkdownDescription: "The name of the NIC.",
},
"description": schema.StringAttribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "Description of the NIC.",
},
"interface": schema.StringAttribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "Interface type.",
},
"driver": schema.StringAttribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "NIC driver.",
},
"model": schema.StringAttribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "NIC model.",
},
"vendor": schema.StringAttribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "NIC vendor.",
},
"port": schema.Int32Attribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "Port number.",
},
"enabled": schema.BoolAttribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "Whether the NIC is enabled.",
},
"vnet": schema.Int32Attribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "The VNet ID to connect to.",
},
"macaddress": schema.StringAttribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "MAC address.",
},
"ipaddress": schema.StringAttribute{
Computed:            true,
MarkdownDescription: "Assigned IP address.",
},
"assign_ipaddress": schema.BoolAttribute{
Optional:            true,
MarkdownDescription: "Whether to assign a static IP address.",
},
"asset": schema.StringAttribute{
Optional:            true,
Computed:            true,
MarkdownDescription: "Asset identification.",
},
"auto_create_vnet": schema.BoolAttribute{
Optional:            true,
MarkdownDescription: "Automatically create a new internal network for this NIC.",
},
"vnet_uplink": schema.Int32Attribute{
Optional:            true,
MarkdownDescription: "Uplink network ID for auto-created VNet.",
},
},
}
}

func (r *NICResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
if req.ProviderData == nil {
return
}
client, ok := req.ProviderData.(*vergeio.Client)
if !ok {
resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *vergeio.Client, got: %T", req.ProviderData))
return
}
r.nicApi = NewNICApi(client)
}

func (r *NICResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
var data StandaloneNICResourceModel
resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
if resp.Diagnostics.HasError() {
return
}

nicData := &nicResourceModel{
Machine:         data.Machine,
Name:            data.Name,
Description:     data.Description,
Interface:       data.Interface,
Driver:          data.Driver,
Model:           data.Model,
Vendor:          data.Vendor,
Port:            data.Port,
Enabled:         data.Enabled,
VNET:            data.VNET,
MAC:             data.MAC,
AssignIPAddress: data.AssignIPAddress,
Asset:           data.Asset,
AutoCreateVNet:  data.AutoCreateVNet,
VNetUplink:      data.VNetUplink,
}

// We need a VM name for auto_create_vnet, but for standalone we can just use the machine ID or NIC name
vmName := data.Name.ValueString() 

if err := r.nicApi.createNIC(ctx, nicData, vmName); err != nil {
resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to create NIC: %s", err))
return
}

data.Id = nicData.Id
data.Name = nicData.Name
data.Description = nicData.Description
data.Interface = nicData.Interface
data.Driver = nicData.Driver
data.Model = nicData.Model
data.Vendor = nicData.Vendor
data.Port = nicData.Port
data.Enabled = nicData.Enabled
data.VNET = nicData.VNET
data.MAC = nicData.MAC
data.Asset = nicData.Asset
data.IPAddress = nicData.IPAddress

resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NICResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
var data StandaloneNICResourceModel
resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
if resp.Diagnostics.HasError() {
return
}

nicData := &nicResourceModel{
Id: data.Id,
}

if err := r.nicApi.readNIC(ctx, nicData); err != nil {
resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to read NIC: %s", err))
return
}

data.Machine = nicData.Machine
data.Name = nicData.Name
data.Description = nicData.Description
data.Interface = nicData.Interface
data.Driver = nicData.Driver
data.Model = nicData.Model
data.Vendor = nicData.Vendor
data.Port = nicData.Port
data.Enabled = nicData.Enabled
data.VNET = nicData.VNET
data.MAC = nicData.MAC
data.Asset = nicData.Asset

resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NICResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
var plan, state StandaloneNICResourceModel
resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
if resp.Diagnostics.HasError() {
return
}

planNICData := &nicResourceModel{
Id:              state.Id,
Machine:         plan.Machine,
Name:            plan.Name,
Description:     plan.Description,
Interface:       plan.Interface,
Driver:          plan.Driver,
Model:           plan.Model,
Vendor:          plan.Vendor,
Port:            plan.Port,
Enabled:         plan.Enabled,
VNET:            plan.VNET,
MAC:             plan.MAC,
AssignIPAddress: plan.AssignIPAddress,
Asset:           plan.Asset,
}

stateNICData := &nicResourceModel{
Id: state.Id,
}

if err := r.nicApi.updateNIC(ctx, planNICData, stateNICData); err != nil {
resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to update NIC: %s", err))
return
}

plan.Id = stateNICData.Id
plan.MAC = stateNICData.MAC

resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NICResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
var data StandaloneNICResourceModel
resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
if resp.Diagnostics.HasError() {
return
}

nicData := &nicResourceModel{
Id: data.Id,
}

vmId := types.StringValue(fmt.Sprintf("%d", data.Machine.ValueInt32()))

if err := r.nicApi.deleteNIC(ctx, nicData, vmId); err != nil {
resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to delete NIC: %s", err))
return
}
}
