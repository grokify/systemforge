package delegation

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/grokify/coreforge/identity/ent"
	"github.com/grokify/coreforge/identity/ent/agent"
	entprincipal "github.com/grokify/coreforge/identity/ent/principal"
	"github.com/grokify/coreforge/identity/principal"
)

// DefaultService implements the Service interface.
type DefaultService struct {
	client *ent.Client
}

// NewService creates a new DelegationService.
func NewService(client *ent.Client) Service {
	return &DefaultService{client: client}
}

// CreateDelegation creates a delegation from one principal to another.
func (s *DefaultService) CreateDelegation(ctx context.Context, input CreateInput) (*Link, error) {
	// Get the delegator
	delegator, err := s.client.Principal.Query().
		Where(entprincipal.ID(input.DelegatorID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("delegator not found: %w", err)
	}

	// Get the delegate
	delegate, err := s.client.Principal.Query().
		Where(entprincipal.ID(input.DelegateID)).
		WithAgent().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("delegate not found: %w", err)
	}

	// Verify delegate is an agent
	if delegate.Type != entprincipal.TypeAgent {
		return nil, fmt.Errorf("delegate must be an agent principal")
	}

	// Verify delegator can delegate
	canDelegate, ok := delegator.Capabilities["can_delegate"]
	if !ok || !canDelegate {
		return nil, fmt.Errorf("delegator cannot delegate")
	}

	// Update agent's delegating_principal_id
	if delegate.Edges.Agent != nil {
		_, err = s.client.Agent.UpdateOne(delegate.Edges.Agent).
			SetDelegatingPrincipalID(input.DelegatorID).
			SetCapabilityConstraints(input.Constraints.AllowedCapabilities).
			SetResourceConstraints(input.Constraints.AllowedResources).
			SetRequiresConfirmation(input.Constraints.RequiresConfirmation).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to update agent delegation: %w", err)
		}

		// Update max token lifetime if specified
		if input.Constraints.MaxTokenLifetime > 0 {
			maxLifetimeSecs := int(input.Constraints.MaxTokenLifetime.Seconds())
			_, err = s.client.Agent.UpdateOne(delegate.Edges.Agent).
				SetMaxTokenLifetime(maxLifetimeSecs).
				Save(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to update agent token lifetime: %w", err)
			}
		}
	}

	return &Link{
		PrincipalID:   input.DelegateID,
		PrincipalType: principal.Type(delegate.Type.String()),
		DisplayName:   delegate.DisplayName,
		Constraints:   input.Constraints,
		GrantedAt:     time.Now(),
	}, nil
}

// GetChain retrieves the full delegation chain for a principal.
func (s *DefaultService) GetChain(ctx context.Context, principalID uuid.UUID) (*Chain, error) {
	// Build chain by walking up delegation tree
	var links []Link
	currentID := principalID
	visited := make(map[uuid.UUID]bool)

	for {
		if visited[currentID] {
			return nil, fmt.Errorf("circular delegation detected")
		}
		visited[currentID] = true

		p, err := s.client.Principal.Query().
			Where(entprincipal.ID(currentID)).
			WithAgent().
			Only(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get principal %s: %w", currentID, err)
		}

		// Build constraints from agent if applicable
		constraints := Constraints{}
		if p.Edges.Agent != nil {
			agent := p.Edges.Agent
			constraints.AllowedCapabilities = agent.CapabilityConstraints
			constraints.AllowedResources = agent.ResourceConstraints
			constraints.RequiresConfirmation = agent.RequiresConfirmation
			constraints.MaxTokenLifetime = time.Duration(agent.MaxTokenLifetime) * time.Second
		}

		link := Link{
			PrincipalID:   p.ID,
			PrincipalType: principal.Type(p.Type.String()),
			DisplayName:   p.DisplayName,
			Constraints:   constraints,
			GrantedAt:     p.CreatedAt,
		}

		// Prepend to links (we're walking up, so reverse order)
		links = append([]Link{link}, links...)

		// Check for parent delegation
		if p.Edges.Agent != nil && p.Edges.Agent.DelegatingPrincipalID != nil {
			currentID = *p.Edges.Agent.DelegatingPrincipalID
		} else {
			// No more parent, we've reached the root
			break
		}
	}

	chain := &Chain{Links: links}
	chain.EffectiveConstraints = s.ComputeEffectiveConstraints(chain)

	return chain, nil
}

// ValidateAction checks if a principal can perform an action on a resource.
func (s *DefaultService) ValidateAction(ctx context.Context, principalID uuid.UUID, action, resource string) (*ActionValidation, error) {
	chain, err := s.GetChain(ctx, principalID)
	if err != nil {
		return nil, err
	}

	// No delegation chain means direct access
	if chain.Depth() <= 1 {
		return &ActionValidation{
			Allowed: true,
			Reason:  "direct principal access",
		}, nil
	}

	effective := chain.EffectiveConstraints

	// Check expiration
	if effective.ExpiresAt != nil && effective.ExpiresAt.Before(time.Now()) {
		return &ActionValidation{
			Allowed: false,
			Reason:  "delegation expired",
		}, nil
	}

	// Check allowed actions
	if len(effective.AllowedActions) > 0 {
		actionAllowed := false
		for _, a := range effective.AllowedActions {
			if a == "*" || a == action {
				actionAllowed = true
				break
			}
		}
		if !actionAllowed {
			return &ActionValidation{
				Allowed: false,
				Reason:  fmt.Sprintf("action %q not in allowed actions", action),
			}, nil
		}
	}

	// Check allowed resources using glob patterns
	if len(effective.AllowedResources) > 0 {
		resourceAllowed := false
		for _, pattern := range effective.AllowedResources {
			if pattern == "*" {
				resourceAllowed = true
				break
			}
			matched, err := filepath.Match(pattern, resource)
			if err == nil && matched {
				resourceAllowed = true
				break
			}
		}
		if !resourceAllowed {
			return &ActionValidation{
				Allowed: false,
				Reason:  fmt.Sprintf("resource %q not in allowed resources", resource),
			}, nil
		}
	}

	return &ActionValidation{
		Allowed:      true,
		Confirmation: effective.RequiresConfirmation,
	}, nil
}

// RevokeDelegation revokes a delegation.
func (s *DefaultService) RevokeDelegation(ctx context.Context, delegatorID, delegateID uuid.UUID, reason string) error {
	// Get the delegate
	delegate, err := s.client.Principal.Query().
		Where(entprincipal.ID(delegateID)).
		WithAgent().
		Only(ctx)
	if err != nil {
		return fmt.Errorf("delegate not found: %w", err)
	}

	if delegate.Edges.Agent == nil {
		return fmt.Errorf("delegate is not an agent")
	}

	// Verify the delegator matches
	if delegate.Edges.Agent.DelegatingPrincipalID == nil ||
		*delegate.Edges.Agent.DelegatingPrincipalID != delegatorID {
		return fmt.Errorf("delegator mismatch")
	}

	// Clear the delegation
	_, err = s.client.Agent.UpdateOne(delegate.Edges.Agent).
		ClearDelegatingPrincipalID().
		SetCapabilityConstraints([]string{}).
		SetResourceConstraints([]string{}).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to revoke delegation: %w", err)
	}

	return nil
}

// ListDelegations lists all delegations granted by a principal.
func (s *DefaultService) ListDelegations(ctx context.Context, delegatorID uuid.UUID) ([]*Link, error) {
	// Find all agents that have this principal as their delegator
	agents, err := s.client.Agent.Query().
		Where(agent.DelegatingPrincipalIDEQ(delegatorID)).
		WithPrincipal().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list delegations: %w", err)
	}

	links := make([]*Link, 0, len(agents))
	for _, agent := range agents {
		if agent.Edges.Principal == nil {
			continue
		}
		p := agent.Edges.Principal
		links = append(links, &Link{
			PrincipalID:   p.ID,
			PrincipalType: principal.Type(p.Type.String()),
			DisplayName:   p.DisplayName,
			Constraints: Constraints{
				AllowedCapabilities:  agent.CapabilityConstraints,
				AllowedResources:     agent.ResourceConstraints,
				RequiresConfirmation: agent.RequiresConfirmation,
				MaxTokenLifetime:     time.Duration(agent.MaxTokenLifetime) * time.Second,
			},
			GrantedAt: p.CreatedAt,
		})
	}

	return links, nil
}

// ComputeEffectiveConstraints computes the effective constraints
// by intersecting constraints through the delegation chain.
func (s *DefaultService) ComputeEffectiveConstraints(chain *Chain) Constraints {
	if chain == nil || len(chain.Links) == 0 {
		return Constraints{}
	}

	// Start with root's constraints (usually empty/unrestricted)
	effective := chain.Links[0].Constraints

	// Intersect with each subsequent link's constraints
	for i := 1; i < len(chain.Links); i++ {
		link := chain.Links[i]
		effective = intersectConstraints(effective, link.Constraints)
	}

	return effective
}

// intersectConstraints returns the intersection of two constraint sets.
func intersectConstraints(a, b Constraints) Constraints {
	result := Constraints{
		RequiresConfirmation: a.RequiresConfirmation || b.RequiresConfirmation,
	}

	// Intersect allowed capabilities
	if len(a.AllowedCapabilities) > 0 && len(b.AllowedCapabilities) > 0 {
		result.AllowedCapabilities = intersectStrings(a.AllowedCapabilities, b.AllowedCapabilities)
	} else if len(b.AllowedCapabilities) > 0 {
		result.AllowedCapabilities = b.AllowedCapabilities
	} else {
		result.AllowedCapabilities = a.AllowedCapabilities
	}

	// Intersect allowed scopes
	if len(a.AllowedScopes) > 0 && len(b.AllowedScopes) > 0 {
		result.AllowedScopes = intersectStrings(a.AllowedScopes, b.AllowedScopes)
	} else if len(b.AllowedScopes) > 0 {
		result.AllowedScopes = b.AllowedScopes
	} else {
		result.AllowedScopes = a.AllowedScopes
	}

	// Intersect allowed resources
	if len(a.AllowedResources) > 0 && len(b.AllowedResources) > 0 {
		result.AllowedResources = intersectStrings(a.AllowedResources, b.AllowedResources)
	} else if len(b.AllowedResources) > 0 {
		result.AllowedResources = b.AllowedResources
	} else {
		result.AllowedResources = a.AllowedResources
	}

	// Intersect allowed actions
	if len(a.AllowedActions) > 0 && len(b.AllowedActions) > 0 {
		result.AllowedActions = intersectStrings(a.AllowedActions, b.AllowedActions)
	} else if len(b.AllowedActions) > 0 {
		result.AllowedActions = b.AllowedActions
	} else {
		result.AllowedActions = a.AllowedActions
	}

	// Take minimum token lifetime
	if a.MaxTokenLifetime > 0 && b.MaxTokenLifetime > 0 {
		if a.MaxTokenLifetime < b.MaxTokenLifetime {
			result.MaxTokenLifetime = a.MaxTokenLifetime
		} else {
			result.MaxTokenLifetime = b.MaxTokenLifetime
		}
	} else if b.MaxTokenLifetime > 0 {
		result.MaxTokenLifetime = b.MaxTokenLifetime
	} else {
		result.MaxTokenLifetime = a.MaxTokenLifetime
	}

	// Take earliest expiration
	if a.ExpiresAt != nil && b.ExpiresAt != nil {
		if a.ExpiresAt.Before(*b.ExpiresAt) {
			result.ExpiresAt = a.ExpiresAt
		} else {
			result.ExpiresAt = b.ExpiresAt
		}
	} else if b.ExpiresAt != nil {
		result.ExpiresAt = b.ExpiresAt
	} else {
		result.ExpiresAt = a.ExpiresAt
	}

	return result
}

// intersectStrings returns strings present in both slices.
func intersectStrings(a, b []string) []string {
	set := make(map[string]bool)
	for _, s := range a {
		set[s] = true
	}

	var result []string
	for _, s := range b {
		if set[s] {
			result = append(result, s)
		}
	}
	return result
}
