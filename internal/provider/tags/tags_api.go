// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

package tags

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"terraform-provider-vergeio/internal/provider/vergeio"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	TagsEndpoint           = "api/v4/tags"
	TagMembersEndpoint     = "api/v4/tag_members"
	TagCategoriesEndpoint  = "api/v4/tag_categories"
)

var _ vergeio.IClient = &TagsApi{}

func NewTagsApi(c *vergeio.Client) *TagsApi {
	return &TagsApi{
		name:   "Tags Api",
		client: c,
	}
}

type TagsApi struct {
	name   string
	client *vergeio.Client
}

func (ta *TagsApi) Name() string {
	return ta.name
}

type TagAPIModel struct {
	Key  int    `json:"$key"`
	Name string `json:"name"`
}

type TagMemberAPIModel struct {
	Key    interface{} `json:"$key,omitempty"`
	Tag    int         `json:"tag"`
	Member string      `json:"member"`
}

// Read tags from the API.
func (ta *TagsApi) readTags(ctx context.Context, data *TagsDataSourceModel) error {
	tflog.Debug(ctx, "Reading tags data")

	// Prepare options for API call
	options := &vergeio.Options{
		Fields: "most",
	}

	// Add name filter if specified
	if !data.Filter.IsNull() && !data.Filter.IsUnknown() {
		filterName := data.Filter.ValueString()
		if filterName != "" {
			options.Filter = fmt.Sprintf("name eq '%s'", filterName)
		}
	}

	apiResp, err := ta.client.Get(TagsEndpoint, options)

	// Error checking with version-aware handling
	if err != nil {
		// Check if this is a 404 error from the client
		if apiError, ok := err.(vergeio.Error); ok && apiError.StatusCode == 404 {
			return fmt.Errorf(vergeio.ErrEndpointV26, "tags")
		}
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	defer apiResp.Body.Close()

	// First check for 404 - version compatibility issue
	if apiResp.StatusCode == 404 {
		return fmt.Errorf(vergeio.ErrEndpointV26, "tags")
	}

	// Check for any other non-200 status code
	if apiResp.StatusCode != 200 {
		return fmt.Errorf("API returned status code %d", apiResp.StatusCode)
	}

	tflog.Debug(ctx, fmt.Sprintf("Read the tags resource %v", apiResp.StatusCode))

	// Read response body
	body, err := io.ReadAll(apiResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Response body: %s", string(body)))

	// Decode the API response
	var tagsAPIResp []TagAPIModel
	if err := json.Unmarshal(body, &tagsAPIResp); err != nil {
		return fmt.Errorf("invalid format received for tags: %w", err)
	}

	// Convert API response to Terraform types
	var tagsList []TagModel
	for _, tag := range tagsAPIResp {
		tagModel := TagModel{
			Key:  types.Int32Value(int32(tag.Key)),
			Name: types.StringValue(tag.Name),
		}
		tagsList = append(tagsList, tagModel)
	}

	// Set the tags in the data model
	data.Tags = tagsList

	tflog.Debug(ctx, fmt.Sprintf("Successfully converted %d tags to resource", len(tagsList)))

	return nil
}

// checkEndpointAvailability tests if an endpoint is available (for version compatibility)
func (ta *TagsApi) checkEndpointAvailability(ctx context.Context, endpoint string) error {
	tflog.Debug(ctx, fmt.Sprintf("Checking availability of endpoint: %s", endpoint))

	// Make a simple GET request to check if endpoint exists
	// We expect either 200 (success) or 40x (endpoint exists but other error)
	// We only care about catching endpoint not found (version issue)
	options := &vergeio.Options{
		Limit: "1", // Minimal response
	}

	apiResp, err := ta.client.Get(endpoint, options)

	// If we get an error from the client, check if it's endpoint-related
	if err != nil {
		if apiError, ok := err.(vergeio.Error); ok && apiError.StatusCode == 404 {
			// Check if the error message indicates endpoint doesn't exist
			if strings.Contains(apiError.VergeError, "not found") && strings.Contains(apiError.Endpoint, endpoint) {
				return fmt.Errorf(vergeio.ErrEndpointV26, strings.TrimPrefix(endpoint, "api/v4/"))
			}
		}
		// Other errors are not version-related, endpoint might still exist
		return nil
	}

	// If we got a response (even non-200), the endpoint exists
	if apiResp != nil {
		apiResp.Body.Close()
	}

	tflog.Debug(ctx, fmt.Sprintf("Endpoint %s is available", endpoint))
	return nil
}

// Create a tag member assignment.
func (ta *TagsApi) createTagMember(ctx context.Context, data *TagMemberResourceModel) error {
	tflog.Debug(ctx, "Creating tag member")

	// First check if the tag_members endpoint is available (version check)
	if err := ta.checkEndpointAvailability(ctx, TagMembersEndpoint); err != nil {
		return err
	}

	// Prepare payload
	payload := TagMemberAPIModel{
		Tag:    int(data.TagId.ValueInt32()),
		Member: data.Member.ValueString(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal tag member payload: %w", err)
	}

	apiResp, err := ta.client.Post(TagMembersEndpoint, bytes.NewBuffer(payloadBytes))

	// Error checking - endpoint is available, so 404s are resource-specific
	if err != nil {
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	defer apiResp.Body.Close()

	// Check for non-success status codes
	if apiResp.StatusCode != 200 && apiResp.StatusCode != 201 {
		return fmt.Errorf("API returned status code %d", apiResp.StatusCode)
	}

	// Read response body to get the created resource
	body, err := io.ReadAll(apiResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Create response body: %s", string(body)))

	// Parse response to get the key
	var createdTagMember TagMemberAPIModel
	if err := json.Unmarshal(body, &createdTagMember); err != nil {
		return fmt.Errorf("invalid format received for created tag member: %w", err)
	}

	// Set the ID from the response
	var keyStr string
	switch v := createdTagMember.Key.(type) {
	case int:
		keyStr = fmt.Sprintf("%d", v)
	case string:
		keyStr = v
	case float64:
		keyStr = fmt.Sprintf("%.0f", v)
	default:
		return fmt.Errorf("unexpected key type: %T", v)
	}
	data.Id = types.StringValue(keyStr)

	tflog.Debug(ctx, fmt.Sprintf("Successfully created tag member with key %s", keyStr))

	return nil
}

// Read tag member from the API.
func (ta *TagsApi) readTagMember(ctx context.Context, data *TagMemberResourceModel) error {
	tflog.Debug(ctx, fmt.Sprintf("Reading tag member with ID %s", data.Id.ValueString()))

	// First check if the tag_members endpoint is available (version check)
	if err := ta.checkEndpointAvailability(ctx, TagMembersEndpoint); err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/%s", TagMembersEndpoint, data.Id.ValueString())
	apiResp, err := ta.client.Get(endpoint, nil)

	// Error checking - endpoint is available, so 404s are resource-specific
	if err != nil {
		if apiError, ok := err.(vergeio.Error); ok && apiError.StatusCode == 404 {
			// Resource-specific not found
			return fmt.Errorf("tag member not found")
		}
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	defer apiResp.Body.Close()

	// Handle 404 from response (resource not found, endpoint exists)
	if apiResp.StatusCode == 404 {
		return fmt.Errorf("tag member not found")
	}

	// Check for any other non-200 status code
	if apiResp.StatusCode != 200 {
		return fmt.Errorf("API returned status code %d", apiResp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(apiResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Response body: %s", string(body)))

	// Decode the API response
	var tagMemberAPIResp TagMemberAPIModel
	if err := json.Unmarshal(body, &tagMemberAPIResp); err != nil {
		return fmt.Errorf("invalid format received for tag member: %w", err)
	}

	// Update the model with API data
	var keyStr string
	switch v := tagMemberAPIResp.Key.(type) {
	case int:
		keyStr = fmt.Sprintf("%d", v)
	case string:
		keyStr = v
	case float64:
		keyStr = fmt.Sprintf("%.0f", v)
	default:
		return fmt.Errorf("unexpected key type: %T", v)
	}
	data.Id = types.StringValue(keyStr)
	data.TagId = types.Int32Value(int32(tagMemberAPIResp.Tag))
	data.Member = types.StringValue(tagMemberAPIResp.Member)

	tflog.Debug(ctx, "Successfully read tag member from API")

	return nil
}

// Update tag member.
func (ta *TagsApi) updateTagMember(ctx context.Context, data *TagMemberResourceModel) error {
	tflog.Debug(ctx, fmt.Sprintf("Updating tag member with ID %s", data.Id.ValueString()))

	// First check if the tag_members endpoint is available (version check)
	if err := ta.checkEndpointAvailability(ctx, TagMembersEndpoint); err != nil {
		return err
	}

	// Prepare payload
	payload := TagMemberAPIModel{
		Tag:    int(data.TagId.ValueInt32()),
		Member: data.Member.ValueString(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal tag member payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/%s", TagMembersEndpoint, data.Id.ValueString())
	apiResp, err := ta.client.Put(endpoint, bytes.NewBuffer(payloadBytes))

	// Error checking - endpoint is available, so 404s are resource-specific
	if err != nil {
		if apiError, ok := err.(vergeio.Error); ok && apiError.StatusCode == 404 {
			return fmt.Errorf("tag member not found")
		}
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	defer apiResp.Body.Close()

	// Handle 404 from response (resource not found, endpoint exists)
	if apiResp.StatusCode == 404 {
		return fmt.Errorf("tag member not found")
	}

	// Check for any other non-200 status code
	if apiResp.StatusCode != 200 {
		return fmt.Errorf("API returned status code %d", apiResp.StatusCode)
	}

	tflog.Debug(ctx, "Successfully updated tag member")

	return nil
}

// Delete tag member.
func (ta *TagsApi) deleteTagMember(ctx context.Context, data *TagMemberResourceModel) error {
	tflog.Debug(ctx, fmt.Sprintf("Deleting tag member with ID %s", data.Id.ValueString()))

	// First check if the tag_members endpoint is available (version check)
	if err := ta.checkEndpointAvailability(ctx, TagMembersEndpoint); err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/%s", TagMembersEndpoint, data.Id.ValueString())
	apiResp, err := ta.client.Delete(endpoint)

	// Error checking - endpoint is available, so 404s are resource-specific
	if err != nil {
		if apiError, ok := err.(vergeio.Error); ok && apiError.StatusCode == 404 {
			// Resource not found during deletion - treat as success since it's gone
			tflog.Debug(ctx, "Tag member not found during deletion (may already be deleted)")
			return nil
		}
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	defer apiResp.Body.Close()

	// Handle 404 from response (resource already deleted, endpoint exists)
	if apiResp.StatusCode == 404 {
		tflog.Debug(ctx, "Tag member not found during deletion (may already be deleted)")
		return nil // Treat as success since resource is gone
	}

	// Check for any other non-success status code
	if apiResp.StatusCode != 200 && apiResp.StatusCode != 204 {
		return fmt.Errorf("API returned status code %d", apiResp.StatusCode)
	}

	tflog.Debug(ctx, "Successfully deleted tag member")

	return nil
}

// TagAPIResourceModel represents the API model for tags
type TagAPIResourceModel struct {
	Key         interface{} `json:"$key,omitempty"`
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	Category    int         `json:"category,omitempty"`
}

// Create a tag.
func (ta *TagsApi) createTag(ctx context.Context, data *TagResourceModel) error {
	tflog.Debug(ctx, "Creating tag")

	// First check if the tags endpoint is available (version check)
	if err := ta.checkEndpointAvailability(ctx, TagsEndpoint); err != nil {
		return err
	}

	// Prepare payload
	payload := TagAPIResourceModel{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
	}
	if !data.Category.IsNull() && !data.Category.IsUnknown() {
		payload.Category = int(data.Category.ValueInt32())
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal tag payload: %w", err)
	}

	apiResp, err := ta.client.Post(TagsEndpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode != 200 && apiResp.StatusCode != 201 {
		return fmt.Errorf("API returned status code %d", apiResp.StatusCode)
	}

	// Read response body to get the created resource
	body, err := io.ReadAll(apiResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Create response body: %s", string(body)))

	// Parse response to get the key
	var createdTag TagAPIResourceModel
	if err := json.Unmarshal(body, &createdTag); err != nil {
		return fmt.Errorf("invalid format received for created tag: %w", err)
	}

	// Set the ID from the response
	keyStr := fmt.Sprintf("%v", createdTag.Key)
	data.Id = types.StringValue(keyStr)

	tflog.Debug(ctx, fmt.Sprintf("Successfully created tag with key %s", keyStr))

	// Read back to get full data
	return ta.readTag(ctx, data)
}

// Read tag from the API.
func (ta *TagsApi) readTag(ctx context.Context, data *TagResourceModel) error {
	tflog.Debug(ctx, fmt.Sprintf("Reading tag with ID %s", data.Id.ValueString()))

	// First check if the tags endpoint is available (version check)
	if err := ta.checkEndpointAvailability(ctx, TagsEndpoint); err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/%s", TagsEndpoint, data.Id.ValueString())
	apiResp, err := ta.client.Get(endpoint, &vergeio.Options{Fields: "most"})

	if err != nil {
		if apiError, ok := err.(vergeio.Error); ok && apiError.StatusCode == 404 {
			data.Id = types.StringNull()
			return nil
		}
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode == 404 {
		data.Id = types.StringNull()
		return nil
	}

	if apiResp.StatusCode != 200 {
		return fmt.Errorf("API returned status code %d", apiResp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(apiResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Response body: %s", string(body)))

	// Decode the API response
	var tagAPIResp TagAPIResourceModel
	if err := json.Unmarshal(body, &tagAPIResp); err != nil {
		return fmt.Errorf("invalid format received for tag: %w", err)
	}

	// Update the model with API data
	keyStr := fmt.Sprintf("%v", tagAPIResp.Key)
	data.Id = types.StringValue(keyStr)
	data.Name = types.StringValue(tagAPIResp.Name)
	
	// Preserve null for description if API returns empty string and plan had null
	if tagAPIResp.Description == "" && data.Description.IsNull() {
		// Keep it null
	} else {
		data.Description = types.StringValue(tagAPIResp.Description)
	}
	
	if tagAPIResp.Category > 0 {
		data.Category = types.Int32Value(int32(tagAPIResp.Category))
	}

	tflog.Debug(ctx, "Successfully read tag from API")

	return nil
}

// Update tag.
func (ta *TagsApi) updateTag(ctx context.Context, data *TagResourceModel) error {
	tflog.Debug(ctx, fmt.Sprintf("Updating tag with ID %s", data.Id.ValueString()))

	// First check if the tags endpoint is available (version check)
	if err := ta.checkEndpointAvailability(ctx, TagsEndpoint); err != nil {
		return err
	}

	// Prepare payload
	payload := TagAPIResourceModel{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
	}
	if !data.Category.IsNull() && !data.Category.IsUnknown() {
		payload.Category = int(data.Category.ValueInt32())
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal tag payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/%s", TagsEndpoint, data.Id.ValueString())
	apiResp, err := ta.client.Put(endpoint, bytes.NewBuffer(payloadBytes))

	if err != nil {
		if apiError, ok := err.(vergeio.Error); ok && apiError.StatusCode == 404 {
			return fmt.Errorf("tag not found")
		}
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode == 404 {
		return fmt.Errorf("tag not found")
	}

	if apiResp.StatusCode != 200 {
		return fmt.Errorf("API returned status code %d", apiResp.StatusCode)
	}

	tflog.Debug(ctx, "Successfully updated tag")

	return ta.readTag(ctx, data)
}

// Delete tag.
func (ta *TagsApi) deleteTag(ctx context.Context, data *TagResourceModel) error {
	tflog.Debug(ctx, fmt.Sprintf("Deleting tag with ID %s", data.Id.ValueString()))

	// First check if the tags endpoint is available (version check)
	if err := ta.checkEndpointAvailability(ctx, TagsEndpoint); err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/%s", TagsEndpoint, data.Id.ValueString())
	apiResp, err := ta.client.Delete(endpoint)

	if err != nil {
		if apiError, ok := err.(vergeio.Error); ok && apiError.StatusCode == 404 {
			tflog.Debug(ctx, "Tag not found during deletion (may already be deleted)")
			return nil
		}
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode == 404 {
		tflog.Debug(ctx, "Tag not found during deletion (may already be deleted)")
		return nil
	}

	if apiResp.StatusCode != 200 && apiResp.StatusCode != 204 {
		return fmt.Errorf("API returned status code %d", apiResp.StatusCode)
	}

	tflog.Debug(ctx, "Successfully deleted tag")

	return nil
}

// TagCategoryAPIResourceModel represents the API model for tag categories
type TagCategoryAPIResourceModel struct {
	Key                     interface{} `json:"$key,omitempty"`
	Name                    string      `json:"name,omitempty"`
	Description             string      `json:"description,omitempty"`
	SingleTagSelection      bool        `json:"single_tag_selection,omitempty"`
	TaggableVMs             bool        `json:"taggable_vms,omitempty"`
	TaggableVolumes         bool        `json:"taggable_volumes,omitempty"`
	TaggableVnets           bool        `json:"taggable_vnets,omitempty"`
	TaggableVnetRules       bool        `json:"taggable_vnet_rules,omitempty"`
	TaggableTenants         bool        `json:"taggable_tenants,omitempty"`
	TaggableTenantNodes     bool        `json:"taggable_tenant_nodes,omitempty"`
	TaggableUsers           bool        `json:"taggable_users,omitempty"`
	TaggableNodes           bool        `json:"taggable_nodes,omitempty"`
	TaggableClusters        bool        `json:"taggable_clusters,omitempty"`
	TaggableGroups          bool        `json:"taggable_groups,omitempty"`
	TaggableSites           bool        `json:"taggable_sites,omitempty"`
	TaggableVmwareContainers bool       `json:"taggable_vmware_containers,omitempty"`
}

// Create a tag category.
func (ta *TagsApi) createTagCategory(ctx context.Context, data *TagCategoryResourceModel) error {
	tflog.Debug(ctx, "Creating tag category")

	if err := ta.checkEndpointAvailability(ctx, TagCategoriesEndpoint); err != nil {
		return err
	}

	payload := TagCategoryAPIResourceModel{
		Name:                    data.Name.ValueString(),
		Description:             data.Description.ValueString(),
		SingleTagSelection:      data.SingleTagSelection.ValueBool(),
		TaggableVMs:             data.TaggableVMs.ValueBool(),
		TaggableVolumes:         data.TaggableVolumes.ValueBool(),
		TaggableVnets:           data.TaggableVnets.ValueBool(),
		TaggableVnetRules:       data.TaggableVnetRules.ValueBool(),
		TaggableTenants:         data.TaggableTenants.ValueBool(),
		TaggableTenantNodes:     data.TaggableTenantNodes.ValueBool(),
		TaggableUsers:           data.TaggableUsers.ValueBool(),
		TaggableNodes:           data.TaggableNodes.ValueBool(),
		TaggableClusters:        data.TaggableClusters.ValueBool(),
		TaggableGroups:          data.TaggableGroups.ValueBool(),
		TaggableSites:           data.TaggableSites.ValueBool(),
		TaggableVmwareContainers: data.TaggableVmwareContainers.ValueBool(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal tag category payload: %w", err)
	}

	apiResp, err := ta.client.Post(TagCategoriesEndpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode != 200 && apiResp.StatusCode != 201 {
		return fmt.Errorf("API returned status code %d", apiResp.StatusCode)
	}

	body, err := io.ReadAll(apiResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Create response body: %s", string(body)))

	var createdCategory TagCategoryAPIResourceModel
	if err := json.Unmarshal(body, &createdCategory); err != nil {
		return fmt.Errorf("invalid format received for created tag category: %w", err)
	}

	keyStr := fmt.Sprintf("%v", createdCategory.Key)
	data.Id = types.StringValue(keyStr)

	tflog.Debug(ctx, fmt.Sprintf("Successfully created tag category with key %s", keyStr))

	return ta.readTagCategory(ctx, data)
}

// Read tag category from the API.
func (ta *TagsApi) readTagCategory(ctx context.Context, data *TagCategoryResourceModel) error {
	tflog.Debug(ctx, fmt.Sprintf("Reading tag category with ID %s", data.Id.ValueString()))

	if err := ta.checkEndpointAvailability(ctx, TagCategoriesEndpoint); err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/%s", TagCategoriesEndpoint, data.Id.ValueString())
	apiResp, err := ta.client.Get(endpoint, &vergeio.Options{Fields: "most"})

	if err != nil {
		if apiError, ok := err.(vergeio.Error); ok && apiError.StatusCode == 404 {
			data.Id = types.StringNull()
			return nil
		}
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode == 404 {
		data.Id = types.StringNull()
		return nil
	}

	if apiResp.StatusCode != 200 {
		return fmt.Errorf("API returned status code %d", apiResp.StatusCode)
	}

	body, err := io.ReadAll(apiResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Response body: %s", string(body)))

	var categoryAPIResp TagCategoryAPIResourceModel
	if err := json.Unmarshal(body, &categoryAPIResp); err != nil {
		return fmt.Errorf("invalid format received for tag category: %w", err)
	}

	keyStr := fmt.Sprintf("%v", categoryAPIResp.Key)
	data.Id = types.StringValue(keyStr)
	data.Name = types.StringValue(categoryAPIResp.Name)
	
	// Preserve null for description if API returns empty string and plan had null
	if categoryAPIResp.Description == "" && data.Description.IsNull() {
		// Keep it null
	} else {
		data.Description = types.StringValue(categoryAPIResp.Description)
	}
	
	data.SingleTagSelection = types.BoolValue(categoryAPIResp.SingleTagSelection)
	data.TaggableVMs = types.BoolValue(categoryAPIResp.TaggableVMs)
	data.TaggableVolumes = types.BoolValue(categoryAPIResp.TaggableVolumes)
	data.TaggableVnets = types.BoolValue(categoryAPIResp.TaggableVnets)
	data.TaggableVnetRules = types.BoolValue(categoryAPIResp.TaggableVnetRules)
	data.TaggableTenants = types.BoolValue(categoryAPIResp.TaggableTenants)
	data.TaggableTenantNodes = types.BoolValue(categoryAPIResp.TaggableTenantNodes)
	data.TaggableUsers = types.BoolValue(categoryAPIResp.TaggableUsers)
	data.TaggableNodes = types.BoolValue(categoryAPIResp.TaggableNodes)
	data.TaggableClusters = types.BoolValue(categoryAPIResp.TaggableClusters)
	data.TaggableGroups = types.BoolValue(categoryAPIResp.TaggableGroups)
	data.TaggableSites = types.BoolValue(categoryAPIResp.TaggableSites)
	data.TaggableVmwareContainers = types.BoolValue(categoryAPIResp.TaggableVmwareContainers)

	tflog.Debug(ctx, "Successfully read tag category from API")

	return nil
}

// Update tag category.
func (ta *TagsApi) updateTagCategory(ctx context.Context, data *TagCategoryResourceModel) error {
	tflog.Debug(ctx, fmt.Sprintf("Updating tag category with ID %s", data.Id.ValueString()))

	if err := ta.checkEndpointAvailability(ctx, TagCategoriesEndpoint); err != nil {
		return err
	}

	payload := TagCategoryAPIResourceModel{
		Name:                    data.Name.ValueString(),
		Description:             data.Description.ValueString(),
		SingleTagSelection:      data.SingleTagSelection.ValueBool(),
		TaggableVMs:             data.TaggableVMs.ValueBool(),
		TaggableVolumes:         data.TaggableVolumes.ValueBool(),
		TaggableVnets:           data.TaggableVnets.ValueBool(),
		TaggableVnetRules:       data.TaggableVnetRules.ValueBool(),
		TaggableTenants:         data.TaggableTenants.ValueBool(),
		TaggableTenantNodes:     data.TaggableTenantNodes.ValueBool(),
		TaggableUsers:           data.TaggableUsers.ValueBool(),
		TaggableNodes:           data.TaggableNodes.ValueBool(),
		TaggableClusters:        data.TaggableClusters.ValueBool(),
		TaggableGroups:          data.TaggableGroups.ValueBool(),
		TaggableSites:           data.TaggableSites.ValueBool(),
		TaggableVmwareContainers: data.TaggableVmwareContainers.ValueBool(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal tag category payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/%s", TagCategoriesEndpoint, data.Id.ValueString())
	apiResp, err := ta.client.Put(endpoint, bytes.NewBuffer(payloadBytes))

	if err != nil {
		if apiError, ok := err.(vergeio.Error); ok && apiError.StatusCode == 404 {
			return fmt.Errorf("tag category not found")
		}
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode == 404 {
		return fmt.Errorf("tag category not found")
	}

	if apiResp.StatusCode != 200 {
		return fmt.Errorf("API returned status code %d", apiResp.StatusCode)
	}

	tflog.Debug(ctx, "Successfully updated tag category")

	return ta.readTagCategory(ctx, data)
}

// Delete tag category.
func (ta *TagsApi) deleteTagCategory(ctx context.Context, data *TagCategoryResourceModel) error {
	tflog.Debug(ctx, fmt.Sprintf("Deleting tag category with ID %s", data.Id.ValueString()))

	if err := ta.checkEndpointAvailability(ctx, TagCategoriesEndpoint); err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/%s", TagCategoriesEndpoint, data.Id.ValueString())
	apiResp, err := ta.client.Delete(endpoint)

	if err != nil {
		if apiError, ok := err.(vergeio.Error); ok && apiError.StatusCode == 404 {
			tflog.Debug(ctx, "Tag category not found during deletion (may already be deleted)")
			return nil
		}
		return err
	}
	if apiResp == nil {
		return errors.New("missing response from the API")
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode == 404 {
		tflog.Debug(ctx, "Tag category not found during deletion (may already be deleted)")
		return nil
	}

	if apiResp.StatusCode != 200 && apiResp.StatusCode != 204 {
		return fmt.Errorf("API returned status code %d", apiResp.StatusCode)
	}

	tflog.Debug(ctx, "Successfully deleted tag category")

	return nil
}
