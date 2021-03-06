package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/manicminer/hamilton/base"
	"github.com/manicminer/hamilton/models"
)

// GroupsClient performs operations on Groups.
type GroupsClient struct {
	BaseClient base.Client
}

// NewGroupsClient returns a new GroupsClient.
func NewGroupsClient(tenantId string) *GroupsClient {
	return &GroupsClient{
		BaseClient: base.NewClient(base.VersionBeta, tenantId),
	}
}

// List returns a list of Groups, optionally filtered using OData.
func (c *GroupsClient) List(ctx context.Context, filter string) (*[]models.Group, int, error) {
	params := url.Values{}
	if filter != "" {
		params.Add("$filter", filter)
	}
	resp, status, _, err := c.BaseClient.Get(ctx, base.GetHttpRequestInput{
		ValidStatusCodes: []int{http.StatusOK},
		Uri: base.Uri{
			Entity:      "/groups",
			Params:      params,
			HasTenantId: true,
		},
	})
	if err != nil {
		return nil, status, err
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)
	var data struct {
		Groups []models.Group `json:"value"`
	}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, status, err
	}
	return &data.Groups, status, nil
}

// Create creates a new Group.
func (c *GroupsClient) Create(ctx context.Context, group models.Group) (*models.Group, int, error) {
	var status int
	body, err := json.Marshal(group)
	if err != nil {
		return nil, status, err
	}
	resp, status, _, err := c.BaseClient.Post(ctx, base.PostHttpRequestInput{
		Body:             body,
		ValidStatusCodes: []int{http.StatusCreated},
		Uri: base.Uri{
			Entity:      "/groups",
			HasTenantId: true,
		},
	})
	if err != nil {
		return nil, status, err
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)
	var newGroup models.Group
	if err := json.Unmarshal(respBody, &newGroup); err != nil {
		return nil, status, err
	}
	return &newGroup, status, nil
}

// Get retrieves a Group.
func (c *GroupsClient) Get(ctx context.Context, id string) (*models.Group, int, error) {
	resp, status, _, err := c.BaseClient.Get(ctx, base.GetHttpRequestInput{
		ValidStatusCodes: []int{http.StatusOK},
		Uri: base.Uri{
			Entity:      fmt.Sprintf("/groups/%s", id),
			HasTenantId: true,
		},
	})
	if err != nil {
		return nil, status, err
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)
	var group models.Group
	if err := json.Unmarshal(respBody, &group); err != nil {
		return nil, status, err
	}
	return &group, status, nil
}

// Update amends an existing Group.
func (c *GroupsClient) Update(ctx context.Context, group models.Group) (int, error) {
	var status int
	body, err := json.Marshal(group)
	if err != nil {
		return status, err
	}
	_, status, _, err = c.BaseClient.Patch(ctx, base.PatchHttpRequestInput{
		Body:             body,
		ValidStatusCodes: []int{http.StatusNoContent},
		Uri: base.Uri{
			Entity:      fmt.Sprintf("/groups/%s", *group.ID),
			HasTenantId: true,
		},
	})
	if err != nil {
		return status, err
	}
	return status, nil
}

// Delete removes a Group.
func (c *GroupsClient) Delete(ctx context.Context, id string) (int, error) {
	_, status, _, err := c.BaseClient.Delete(ctx, base.DeleteHttpRequestInput{
		ValidStatusCodes: []int{http.StatusNoContent},
		Uri: base.Uri{
			Entity:      fmt.Sprintf("/groups/%s", id),
			HasTenantId: true,
		},
	})
	if err != nil {
		return status, err
	}
	return status, nil
}

// ListMembers retrieves the members of the specified Group.
// id is the object ID of the group.
func (c *GroupsClient) ListMembers(ctx context.Context, id string) (*[]string, int, error) {
	resp, status, _, err := c.BaseClient.Get(ctx, base.GetHttpRequestInput{
		ValidStatusCodes: []int{http.StatusOK},
		Uri: base.Uri{
			Entity:      fmt.Sprintf("/groups/%s/members", id),
			Params:      url.Values{"$select": []string{"id"}},
			HasTenantId: true,
		},
	})
	if err != nil {
		return nil, status, err
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)
	var data struct {
		Members []struct {
			Type string `json:"@odata.type"`
			Id   string `json:"id"`
		} `json:"value"`
	}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, status, err
	}
	ret := make([]string, len(data.Members))
	for i, v := range data.Members {
		ret[i] = v.Id
	}
	return &ret, status, nil
}

// GetMember retrieves a single member of the specified Group.
// groupId is the object ID of the group.
// memberId is the object ID of the member object.
func (c *GroupsClient) GetMember(ctx context.Context, groupId, memberId string) (*string, int, error) {
	resp, status, _, err := c.BaseClient.Get(ctx, base.GetHttpRequestInput{
		ValidStatusCodes: []int{http.StatusOK},
		Uri: base.Uri{
			Entity:      fmt.Sprintf("/groups/%s/members/%s/$ref", groupId, memberId),
			Params:      url.Values{"$select": []string{"id,url"}},
			HasTenantId: true,
		},
	})
	if err != nil {
		return nil, status, err
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)
	var data struct {
		Context string `json:"@odata.context"`
		Type    string `json:"@odata.type"`
		Id      string `json:"id"`
		Url     string `json:"url"`
	}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, status, err
	}
	return &data.Id, status, nil
}

// AddMembers adds a new member to a Group.
// First populate the Members field of the Group using the AppendMember method of the model, then call this method.
func (c *GroupsClient) AddMembers(ctx context.Context, group *models.Group) (int, error) {
	var status int
	// Patching group members support up to 20 members per request
	var memberChunks [][]string
	members := *group.Members
	max := len(members)
	// Chunk into slices of 20 for batching
	for i := 0; i < max; i += 20 {
		end := i + 20
		if end > max {
			end = max
		}
		memberChunks = append(memberChunks, members[i:end])
	}
	for _, members := range memberChunks {
		data := struct {
			Members []string `json:"members@odata.bind"`
		}{
			Members: members,
		}
		body, err := json.Marshal(data)
		if err != nil {
			return status, err
		}
		_, status, _, err = c.BaseClient.Patch(ctx, base.PatchHttpRequestInput{
			Body:             body,
			ValidStatusCodes: []int{http.StatusNoContent},
			Uri: base.Uri{
				Entity:      fmt.Sprintf("/groups/%s", *group.ID),
				HasTenantId: true,
			},
		})
		if err != nil {
			return status, err
		}
	}
	return status, nil
}

// RemoveMembers removes members from a Group.
// groupId is the object ID of the group.
// memberIds is a *[]string containing object IDs of members to remove.
func (c *GroupsClient) RemoveMembers(ctx context.Context, id string, memberIds *[]string) (int, error) {
	var status int
	for _, memberId := range *memberIds {
		// check for membership before attempting deletion
		if _, status, err := c.GetMember(ctx, id, memberId); err != nil {
			if status == http.StatusNotFound {
				continue
			}
			return status, err
		}
		var err error
		_, status, _, err = c.BaseClient.Delete(ctx, base.DeleteHttpRequestInput{
			ValidStatusCodes: []int{http.StatusNoContent},
			Uri: base.Uri{
				Entity:      fmt.Sprintf("/groups/%s/members/%s/$ref", id, memberId),
				HasTenantId: true,
			},
		})
		if err != nil {
			return status, err
		}
	}
	return status, nil
}

// ListOwners retrieves the owners of the specified Group.
// id is the object ID of the group.
func (c *GroupsClient) ListOwners(ctx context.Context, id string) (*[]string, int, error) {
	resp, status, _, err := c.BaseClient.Get(ctx, base.GetHttpRequestInput{
		ValidStatusCodes: []int{http.StatusOK},
		Uri: base.Uri{
			Entity:      fmt.Sprintf("/groups/%s/owners", id),
			Params:      url.Values{"$select": []string{"id"}},
			HasTenantId: true,
		},
	})
	if err != nil {
		return nil, status, err
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)
	var data struct {
		Owners []struct {
			Type string `json:"@odata.type"`
			Id   string `json:"id"`
		} `json:"value"`
	}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, status, err
	}
	ret := make([]string, len(data.Owners))
	for i, v := range data.Owners {
		ret[i] = v.Id
	}
	return &ret, status, nil
}

// GetOwner retrieves a single owner for the specified Group.
// groupId is the object ID of the group.
// ownerId is the object ID of the owning object.
func (c *GroupsClient) GetOwner(ctx context.Context, groupId, ownerId string) (*string, int, error) {
	resp, status, _, err := c.BaseClient.Get(ctx, base.GetHttpRequestInput{
		ValidStatusCodes: []int{http.StatusOK},
		Uri: base.Uri{
			Entity:      fmt.Sprintf("/groups/%s/owners/%s/$ref", groupId, ownerId),
			Params:      url.Values{"$select": []string{"id,url"}},
			HasTenantId: true,
		},
	})
	if err != nil {
		return nil, status, err
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)
	var data struct {
		Context string `json:"@odata.context"`
		Type    string `json:"@odata.type"`
		Id      string `json:"id"`
		Url     string `json:"url"`
	}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, status, err
	}
	return &data.Id, status, nil
}

// AddOwners adds a new owner to a Group.
// First populate the Owners field of the Group using the AppendOwner method of the model, then call this method.
func (c *GroupsClient) AddOwners(ctx context.Context, group *models.Group) (int, error) {
	var status int
	for _, owner := range *group.Owners {
		data := struct {
			Owner string `json:"@odata.id"`
		}{
			Owner: owner,
		}
		body, err := json.Marshal(data)
		if err != nil {
			return status, err
		}
		_, status, _, err = c.BaseClient.Post(ctx, base.PostHttpRequestInput{
			Body:             body,
			ValidStatusCodes: []int{http.StatusNoContent},
			Uri: base.Uri{
				Entity:      fmt.Sprintf("/groups/%s/owners/$ref", *group.ID),
				HasTenantId: true,
			},
		})
		if err != nil {
			return status, err
		}
	}
	return status, nil
}

// RemoveOwners removes owners from a Group.
// groupId is the object ID of the group.
// ownerIds is a *[]string containing object IDs of owners to remove.
func (c *GroupsClient) RemoveOwners(ctx context.Context, id string, ownerIds *[]string) (int, error) {
	var status int
	for _, ownerId := range *ownerIds {
		// check for ownership before attempting deletion
		if _, status, err := c.GetOwner(ctx, id, ownerId); err != nil {
			if status == http.StatusNotFound {
				continue
			}
			return status, err
		}
		var err error
		_, status, _, err = c.BaseClient.Delete(ctx, base.DeleteHttpRequestInput{
			ValidStatusCodes: []int{http.StatusNoContent},
			Uri: base.Uri{
				Entity:      fmt.Sprintf("/groups/%s/owners/%s/$ref", id, ownerId),
				HasTenantId: true,
			},
		})
		if err != nil {
			return status, err
		}
	}
	return status, nil
}
