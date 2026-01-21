package vm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	MachineSnapshotEndpoint = "api/v4/machine_snapshots"
)

type VMSnapshotAPIResourceModel struct {
	Id          interface{} `json:"$key,omitempty"`
	Machine     int32       `json:"machine,omitempty"`
	SnapMachine int32       `json:"snap_machine,omitempty"`
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	Created     int64       `json:"created,omitempty"`
}

type VMSnapshotResponse struct {
	Key      interface{} `json:"$key,omitempty"`
	Response struct {
		VMKey      interface{} `json:"vmkey,omitempty"`
		MachineKey interface{} `json:"machinekey,omitempty"`
	} `json:"response,omitempty"`
}

func (va *VMApi) CreateVMSnapshot(ctx context.Context, data *VMSnapshotResourceModel) error {
	// First, we need to get the machine ID for the VM
	vmModel := &VMResourceModel{
		Id: data.VMId,
	}
	if err := va.readVM(ctx, vmModel); err != nil {
		return fmt.Errorf("error reading VM %s: %s", data.VMId.ValueString(), err)
	}

	apiData := VMSnapshotAPIResourceModel{
		Machine:     vmModel.Machine.ValueInt32(),
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
	}

	encodedBuffer := new(bytes.Buffer)
	if err := json.NewEncoder(encodedBuffer).Encode(apiData); err != nil {
		return err
	}

	apiResp, err := va.client.Post(MachineSnapshotEndpoint, encodedBuffer)
	if err != nil {
		return err
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode != 201 {
		return fmt.Errorf("unexpected status code %d during snapshot creation", apiResp.StatusCode)
	}

	var resp VMSnapshotResponse
	if err := json.NewDecoder(apiResp.Body).Decode(&resp); err != nil {
		return err
	}

	// Helper to convert interface{} key to string
	keyStr := fmt.Sprintf("%v", resp.Key)
	data.Id = types.StringValue(keyStr)

	return va.ReadVMSnapshot(ctx, data)
}

func (va *VMApi) ReadVMSnapshot(ctx context.Context, data *VMSnapshotResourceModel) error {
	apiResp, err := va.client.Get(fmt.Sprintf("%s/%s", MachineSnapshotEndpoint, url.PathEscape(data.Id.ValueString())), nil)
	if err != nil {
		return err
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode == 404 {
		data.Id = types.StringNull()
		return nil
	}
	if apiResp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code %d during snapshot read", apiResp.StatusCode)
	}

	var apiModel VMSnapshotAPIResourceModel
	if err := json.NewDecoder(apiResp.Body).Decode(&apiModel); err != nil {
		return err
	}

	data.Name = types.StringValue(apiModel.Name)
	data.Description = types.StringValue(apiModel.Description)
	data.SnapMachineId = types.Int32Value(apiModel.SnapMachine)
	data.Created = types.Int64Value(apiModel.Created)
	
	// Parent machine ID
	data.MachineId = types.Int32Value(apiModel.Machine)

	return nil
}

func (va *VMApi) UpdateVMSnapshot(ctx context.Context, data *VMSnapshotResourceModel) error {
	apiData := VMSnapshotAPIResourceModel{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
	}

	encodedBuffer := new(bytes.Buffer)
	if err := json.NewEncoder(encodedBuffer).Encode(apiData); err != nil {
		return err
	}

	apiResp, err := va.client.Put(fmt.Sprintf("%s/%s", MachineSnapshotEndpoint, url.PathEscape(data.Id.ValueString())), encodedBuffer)
	if err != nil {
		return err
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code %d during snapshot update", apiResp.StatusCode)
	}

	return va.ReadVMSnapshot(ctx, data)
}

func (va *VMApi) DeleteVMSnapshot(ctx context.Context, data *VMSnapshotResourceModel) error {
	apiResp, err := va.client.Delete(fmt.Sprintf("%s/%s", MachineSnapshotEndpoint, url.PathEscape(data.Id.ValueString())))
	if err != nil {
		return err
	}
	if apiResp.StatusCode != 200 && apiResp.StatusCode != 404 {
		return fmt.Errorf("unexpected status code %d during snapshot deletion", apiResp.StatusCode)
	}
	return nil
}
