package organization

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateOrganization(ctx context.Context, input CreateOrganizationInput) (*Organization, error) {
	org := &Organization{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO organizations (name, description) VALUES ($1, $2)
		 RETURNING id, name, description, created_at, updated_at`,
		input.Name, input.Description,
	).Scan(&org.ID, &org.Name, &org.Description, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create organization: %w", err)
	}
	return org, nil
}

func (r *PostgresRepository) GetOrganizationByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	org := &Organization{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, description, created_at, updated_at FROM organizations WHERE id = $1`, id,
	).Scan(&org.ID, &org.Name, &org.Description, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get organization: %w", err)
	}
	return org, nil
}

func (r *PostgresRepository) CreateMVRU(ctx context.Context, input CreateMVRUInput) (*MVRU, error) {
	boundaryJSON, _ := json.Marshal(input.Boundary)
	configJSON, _ := json.Marshal(input.Config)

	mvru := &MVRU{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO muvrs (organization_id, name, description, boundary, config, parent_id)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, organization_id, name, description, status, boundary, config, parent_id, created_at, updated_at`,
		input.OrganizationID, input.Name, input.Description, boundaryJSON, configJSON, input.ParentID,
	).Scan(&mvru.ID, &mvru.OrganizationID, &mvru.Name, &mvru.Description, &mvru.Status, &boundaryJSON, &configJSON, &mvru.ParentID, &mvru.CreatedAt, &mvru.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create mvru: %w", err)
	}
	json.Unmarshal(boundaryJSON, &mvru.Boundary)
	json.Unmarshal(configJSON, &mvru.Config)
	return mvru, nil
}

func (r *PostgresRepository) GetMVRUByID(ctx context.Context, id uuid.UUID) (*MVRU, error) {
	mvru := &MVRU{}
	var boundaryJSON, configJSON []byte
	err := r.db.QueryRow(ctx,
		`SELECT id, organization_id, name, description, status, boundary, config, parent_id, created_at, updated_at
		 FROM muvrs WHERE id = $1`, id,
	).Scan(&mvru.ID, &mvru.OrganizationID, &mvru.Name, &mvru.Description, &mvru.Status, &boundaryJSON, &configJSON, &mvru.ParentID, &mvru.CreatedAt, &mvru.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get mvru: %w", err)
	}
	json.Unmarshal(boundaryJSON, &mvru.Boundary)
	json.Unmarshal(configJSON, &mvru.Config)
	return mvru, nil
}

func (r *PostgresRepository) ListMVRUs(ctx context.Context, orgID uuid.UUID) ([]MVRU, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, organization_id, name, description, status, boundary, config, parent_id, created_at, updated_at
		 FROM muvrs WHERE organization_id = $1 ORDER BY created_at`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list muvrs: %w", err)
	}
	defer rows.Close()

	var muvrs []MVRU
	for rows.Next() {
		var mvru MVRU
		var boundaryJSON, configJSON []byte
		if err := rows.Scan(&mvru.ID, &mvru.OrganizationID, &mvru.Name, &mvru.Description, &mvru.Status, &boundaryJSON, &configJSON, &mvru.ParentID, &mvru.CreatedAt, &mvru.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan mvru: %w", err)
		}
		json.Unmarshal(boundaryJSON, &mvru.Boundary)
		json.Unmarshal(configJSON, &mvru.Config)
		muvrs = append(muvrs, mvru)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list muvrs iteration: %w", err)
	}
	return muvrs, nil
}

func (r *PostgresRepository) UpdateMVRUStatus(ctx context.Context, id uuid.UUID, status MVRUStatus) error {
	_, err := r.db.Exec(ctx, `UPDATE muvrs SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	if err != nil {
		return fmt.Errorf("update mvru status: %w", err)
	}
	return nil
}

func (r *PostgresRepository) AddMember(ctx context.Context, member MVRUMember) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO mvru_members (mvru_id, user_id, agent_id, role_id) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (mvru_id, COALESCE(user_id, agent_id)) DO UPDATE SET role_id = $4`,
		member.MVRUID, member.UserID, member.AgentID, member.RoleID)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

func (r *PostgresRepository) RemoveMember(ctx context.Context, mvruID, userID, agentID *uuid.UUID) error {
	if userID != nil {
		_, err := r.db.Exec(ctx, `DELETE FROM mvru_members WHERE mvru_id = $1 AND user_id = $2`, mvruID, *userID)
		if err != nil {
			return fmt.Errorf("remove user member: %w", err)
		}
	} else if agentID != nil {
		_, err := r.db.Exec(ctx, `DELETE FROM mvru_members WHERE mvru_id = $1 AND agent_id = $2`, mvruID, *agentID)
		if err != nil {
			return fmt.Errorf("remove agent member: %w", err)
		}
	}
	return nil
}

func (r *PostgresRepository) CreateRelationship(ctx context.Context, rel MVRURelationship) (*MVRURelationship, error) {
	configJSON, _ := json.Marshal(rel.Config)
	err := r.db.QueryRow(ctx,
		`INSERT INTO mvru_relationships (source_mvru_id, target_mvru_id, rel_type, config)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, source_mvru_id, target_mvru_id, rel_type, config, created_at`,
		rel.SourceMVRUID, rel.TargetMVRUID, rel.RelType, configJSON,
	).Scan(&rel.ID, &rel.SourceMVRUID, &rel.TargetMVRUID, &rel.RelType, &configJSON, &rel.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create relationship: %w", err)
	}
	json.Unmarshal(configJSON, &rel.Config)
	return &rel, nil
}

func (r *PostgresRepository) GetOrgChart(ctx context.Context, orgID uuid.UUID) ([]MVRU, error) {
	all, err := r.ListMVRUs(ctx, orgID)
	if err != nil {
		return nil, err
	}

	childMap := make(map[uuid.UUID][]MVRU)
	for _, mv := range all {
		if mv.ParentID != nil {
			childMap[*mv.ParentID] = append(childMap[*mv.ParentID], mv)
		}
	}

	var roots []MVRU
	for _, mv := range all {
		if mv.ParentID == nil {
			mv.Children = childMap[mv.ID]
			roots = append(roots, mv)
		}
	}
	return roots, nil
}
