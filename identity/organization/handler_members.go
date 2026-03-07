package organization

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

// MemberResponse is the API representation of a membership.
type MemberResponse struct {
	ID             string    `json:"id"`
	PrincipalID    string    `json:"principal_id"`
	OrganizationID string    `json:"organization_id"`
	Role           string    `json:"role"`
	Permissions    []string  `json:"permissions,omitempty"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func memberToResponse(m *Membership) *MemberResponse {
	return &MemberResponse{
		ID:             m.ID.String(),
		PrincipalID:    m.PrincipalID.String(),
		OrganizationID: m.OrganizationID.String(),
		Role:           m.Role,
		Permissions:    m.Permissions,
		Active:         m.Active,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

// AddMemberRequest is the request for adding a member.
type AddMemberRequest struct {
	Slug string `path:"slug"`
	Body struct {
		PrincipalID string   `json:"principal_id" required:"true" format:"uuid"`
		Role        string   `json:"role" required:"true" enum:"owner,admin,member"`
		Permissions []string `json:"permissions,omitempty"`
	}
}

// AddMemberResponse is the response for adding a member.
type AddMemberResponse struct {
	Body *MemberResponse
}

// UpdateMemberRequest is the request for updating a member.
type UpdateMemberRequest struct {
	Slug        string `path:"slug"`
	PrincipalID string `path:"principal_id" format:"uuid"`
	Body        struct {
		Role        *string  `json:"role,omitempty" enum:"owner,admin,member"`
		Permissions []string `json:"permissions,omitempty"`
		Active      *bool    `json:"active,omitempty"`
	}
}

// UpdateMemberResponse is the response for updating a member.
type UpdateMemberResponse struct {
	Body *MemberResponse
}

// RemoveMemberInput is the request for removing a member.
type RemoveMemberInput struct {
	Slug        string `path:"slug"`
	PrincipalID string `path:"principal_id" format:"uuid"`
}

// ListMembersInput is the request for listing members.
type ListMembersInput struct {
	Slug string `path:"slug"`
}

// ListMembersOutput is the response for listing members.
type ListMembersOutput struct {
	Body struct {
		Members []*MemberResponse `json:"members"`
	}
}

// GetMemberInput is the request for getting a specific member.
type GetMemberInput struct {
	Slug        string `path:"slug"`
	PrincipalID string `path:"principal_id" format:"uuid"`
}

// GetMemberOutput is the response for getting a member.
type GetMemberOutput struct {
	Body *MemberResponse
}

// registerMemberEndpoints registers membership management endpoints.
func (a *API) registerMemberEndpoints() {
	basePath := a.config.BasePath

	// List Members
	huma.Register(a.huma, huma.Operation{
		OperationID:   "listMembers",
		Method:        http.MethodGet,
		Path:          basePath + "/organizations/{slug}/members",
		Summary:       "List organization members",
		Description:   "Returns all members of an organization",
		Tags:          []string{"Members"},
		DefaultStatus: http.StatusOK,
	}, a.listMembers)

	// Get Member
	huma.Register(a.huma, huma.Operation{
		OperationID:   "getMember",
		Method:        http.MethodGet,
		Path:          basePath + "/organizations/{slug}/members/{principal_id}",
		Summary:       "Get a member",
		Description:   "Returns a specific member of an organization",
		Tags:          []string{"Members"},
		DefaultStatus: http.StatusOK,
	}, a.getMember)

	// Add Member
	huma.Register(a.huma, huma.Operation{
		OperationID:   "addMember",
		Method:        http.MethodPost,
		Path:          basePath + "/organizations/{slug}/members",
		Summary:       "Add a member",
		Description:   "Adds a principal as a member of an organization",
		Tags:          []string{"Members"},
		DefaultStatus: http.StatusCreated,
	}, a.addMember)

	// Update Member
	huma.Register(a.huma, huma.Operation{
		OperationID:   "updateMember",
		Method:        http.MethodPatch,
		Path:          basePath + "/organizations/{slug}/members/{principal_id}",
		Summary:       "Update a member",
		Description:   "Updates a member's role or permissions",
		Tags:          []string{"Members"},
		DefaultStatus: http.StatusOK,
	}, a.updateMember)

	// Remove Member
	huma.Register(a.huma, huma.Operation{
		OperationID:   "removeMember",
		Method:        http.MethodDelete,
		Path:          basePath + "/organizations/{slug}/members/{principal_id}",
		Summary:       "Remove a member",
		Description:   "Removes a member from an organization",
		Tags:          []string{"Members"},
		DefaultStatus: http.StatusNoContent,
	}, a.removeMember)
}

func (a *API) listMembers(ctx context.Context, input *ListMembersInput) (*ListMembersOutput, error) {
	org, err := a.service.GetBySlug(ctx, input.Slug)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}

	members, err := a.service.ListMembers(ctx, org.ID)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	responses := make([]*MemberResponse, len(members))
	for i, m := range members {
		responses[i] = memberToResponse(m)
	}

	output := &ListMembersOutput{}
	output.Body.Members = responses

	return output, nil
}

func (a *API) getMember(ctx context.Context, input *GetMemberInput) (*GetMemberOutput, error) {
	org, err := a.service.GetBySlug(ctx, input.Slug)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}

	principalID, err := uuid.Parse(input.PrincipalID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid principal_id")
	}

	member, err := a.service.GetMembership(ctx, org.ID, principalID)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}

	return &GetMemberOutput{Body: memberToResponse(member)}, nil
}

func (a *API) addMember(ctx context.Context, input *AddMemberRequest) (*AddMemberResponse, error) {
	org, err := a.service.GetBySlug(ctx, input.Slug)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}

	principalID, err := uuid.Parse(input.Body.PrincipalID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid principal_id")
	}

	member, err := a.service.AddMember(ctx, AddMemberInput{
		OrganizationID: org.ID,
		PrincipalID:    principalID,
		Role:           input.Body.Role,
		Permissions:    input.Body.Permissions,
	})
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &AddMemberResponse{Body: memberToResponse(member)}, nil
}

func (a *API) updateMember(ctx context.Context, input *UpdateMemberRequest) (*UpdateMemberResponse, error) {
	org, err := a.service.GetBySlug(ctx, input.Slug)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}

	principalID, err := uuid.Parse(input.PrincipalID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid principal_id")
	}

	membership, err := a.service.GetMembership(ctx, org.ID, principalID)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}

	updated, err := a.service.UpdateMember(ctx, membership.ID, UpdateMemberInput{
		Role:        input.Body.Role,
		Permissions: input.Body.Permissions,
		Active:      input.Body.Active,
	})
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &UpdateMemberResponse{Body: memberToResponse(updated)}, nil
}

func (a *API) removeMember(ctx context.Context, input *RemoveMemberInput) (*struct{}, error) {
	org, err := a.service.GetBySlug(ctx, input.Slug)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}

	principalID, err := uuid.Parse(input.PrincipalID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid principal_id")
	}

	if err := a.service.RemoveMember(ctx, org.ID, principalID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return nil, nil
}

