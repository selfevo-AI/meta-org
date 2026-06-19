package singleorg

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *DBRepository {
	return &DBRepository{db: db}
}

func (r *DBRepository) GetUserProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error) {
	profile := &UserProfile{}
	var defaultOrg pgtype.UUID
	err := r.db.QueryRow(ctx, `
		SELECT id, name, email, account_status, onboarding_status, default_organization_id, created_at, updated_at
		FROM users
		WHERE id = $1
	`, userID).Scan(
		&profile.ID,
		&profile.Name,
		&profile.Email,
		&profile.AccountStatus,
		&profile.OnboardingStatus,
		&defaultOrg,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get user profile: %w", err)
	}
	profile.DefaultOrganizationID = uuidPointer(defaultOrg)
	orgs, err := r.ListOrganizationsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	profile.Organizations = orgs
	profile.OnboardingRequired = false
	return profile, nil
}

func (r *DBRepository) ListOrganizationsForUser(ctx context.Context, userID uuid.UUID) ([]OrganizationAccount, error) {
	rows, err := r.db.Query(ctx, `
		SELECT o.id, o.name, COALESCE(o.description, ''), om.id, om.authority_tier
		FROM organization_memberships om
		JOIN organizations o ON o.id = om.organization_id
		WHERE om.user_id = $1 AND om.status = 'active'
		ORDER BY om.joined_at ASC, o.name
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list user organizations: %w", err)
	}
	defer rows.Close()

	items := []OrganizationAccount{}
	for rows.Next() {
		var item OrganizationAccount
		var membershipID uuid.UUID
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &membershipID, &item.AuthorityTier); err != nil {
			return nil, fmt.Errorf("scan user organization: %w", err)
		}
		item.MembershipID = &membershipID
		item.IsOwner = item.AuthorityTier == AuthorityOwner
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list user organizations iteration: %w", err)
	}
	return items, nil
}

func (r *DBRepository) EnsureSingleOrgForUser(ctx context.Context, userID uuid.UUID) (*OrganizationAccount, error) {
	orgs, err := r.ListOrganizationsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(orgs) > 0 {
		if _, err := r.db.Exec(ctx, `
			UPDATE users
			SET onboarding_status = 'complete',
			    default_organization_id = COALESCE(default_organization_id, $2),
			    updated_at = NOW()
			WHERE id = $1
		`, userID, orgs[0].ID); err != nil {
			return nil, fmt.Errorf("update default organization: %w", err)
		}
		return &orgs[0], nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin single organization bootstrap: %w", err)
	}
	defer tx.Rollback(ctx)

	var org OrganizationAccount
	if err := tx.QueryRow(ctx, `
		INSERT INTO organizations (name, description, created_by)
		VALUES ('Default Organization', 'Single organization workspace', $1)
		RETURNING id, name, COALESCE(description, '')
	`, userID).Scan(&org.ID, &org.Name, &org.Description); err != nil {
		return nil, fmt.Errorf("create default organization: %w", err)
	}

	var deptID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO departments (organization_id, name, code, description, metadata)
		VALUES ($1, 'Default Department', 'DEFAULT', 'Created for single organization mode', '{"system_created":true}'::jsonb)
		RETURNING id
	`, org.ID).Scan(&deptID); err != nil {
		return nil, fmt.Errorf("create default department: %w", err)
	}

	var membershipID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO organization_memberships (
			organization_id, department_id, member_type, user_id, title, authority_tier, status, metadata
		)
		VALUES ($1, $2, 'internal', $3, 'Owner', 'organization_creator', 'active', '{"source":"single_org_bootstrap"}'::jsonb)
		RETURNING id
	`, org.ID, deptID, userID).Scan(&membershipID); err != nil {
		return nil, fmt.Errorf("create owner membership: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE users
		SET onboarding_status = 'complete',
		    default_organization_id = $2,
		    updated_at = NOW()
		WHERE id = $1
	`, userID, org.ID); err != nil {
		return nil, fmt.Errorf("complete single organization user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit single organization bootstrap: %w", err)
	}
	org.MembershipID = &membershipID
	org.AuthorityTier = AuthorityOwner
	org.IsOwner = true
	return &org, nil
}

func (r *DBRepository) GetHumanMembership(ctx context.Context, userID uuid.UUID, orgID uuid.UUID) (*Membership, error) {
	item := &Membership{}
	err := r.db.QueryRow(ctx, `
		SELECT id, organization_id, authority_tier
		FROM organization_memberships
		WHERE organization_id = $1
		  AND user_id = $2
		  AND member_type = 'internal'
		  AND status = 'active'
		ORDER BY joined_at ASC
		LIMIT 1
	`, orgID, userID).Scan(&item.ID, &item.OrganizationID, &item.AuthorityTier)
	if err != nil {
		return nil, fmt.Errorf("get human membership: %w", err)
	}
	return item, nil
}

func (r *DBRepository) GetAgentMembership(ctx context.Context, agentID uuid.UUID, orgID uuid.UUID) (*Membership, error) {
	item := &Membership{}
	err := r.db.QueryRow(ctx, `
		SELECT id, organization_id, authority_tier
		FROM organization_memberships
		WHERE organization_id = $1
		  AND agent_id = $2
		  AND member_type = 'agent'
		  AND status = 'active'
		ORDER BY joined_at ASC
		LIMIT 1
	`, orgID, agentID).Scan(&item.ID, &item.OrganizationID, &item.AuthorityTier)
	if err != nil {
		return nil, fmt.Errorf("get agent membership: %w", err)
	}
	return item, nil
}

func uuidPointer(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	id := uuid.UUID(value.Bytes)
	return &id
}
