package repository

import (
	"context"
	"fmt"
	"time"

	"charm.land/log/v2"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector/models"
)

var _ models.Repository = (*Nodes)(nil)

type Nodes struct {
	db *bun.DB
}

func NewNodes(db *bun.DB) *Nodes {
	return &Nodes{db: db}
}

func (n *Nodes) Migrate(ctx context.Context) error {
	query := n.db.NewCreateTable().
		Model(&models.Node{}).
		IfNotExists()

	_, err := query.Exec(ctx)
	if err != nil {
		log.Debug(query.String())
		log.Error("failed to create node table", "error", err.Error())

		return fmt.Errorf("migrate model: %w", err)
	}

	return nil
}

func (n *Nodes) GetNode(ctx context.Context, id uuid.UUID) (*models.Node, error) {
	var node models.Node

	err := n.db.NewSelect().Model(&node).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed select node with id %s: %w", id, err)
	}

	return &node, nil
}

func (n *Nodes) GetNodes(ctx context.Context, organizationID uuid.UUID) ([]models.Node, error) {
	var nodes []models.Node

	err := n.db.NewSelect().
		Model(&nodes).
		Where("organization_id = ?", organizationID).
		Scan(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes for organization %s: %w", organizationID, err)
	}

	return nodes, nil
}

func (n *Nodes) NewNode(ctx context.Context, id uuid.UUID, name string) (*models.Node, error) {
	now := time.Now()
	model := &models.Node{
		ID:              id,
		Name:            name,
		LastConnection:  &now,
		FirstConnection: now,
	}

	_, err := n.db.NewInsert().Model(model).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	return model, nil
}

func (n *Nodes) ReconnectNode(ctx context.Context, id uuid.UUID) (*models.Node, error) {
	var node models.Node

	_, err := n.db.NewUpdate().
		Model(&node).
		Where("id = ?", id).
		Set("last_connection = ?", time.Now()).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to reconnect node %s: %w", id, err)
	}

	return &node, nil
}

func (n *Nodes) UpdateLastConnection(ctx context.Context, id uuid.UUID) (*models.Node, error) {
	var node models.Node

	_, err := n.db.NewUpdate().
		Model(&node).
		Where("id = ?", id).
		Set("last_connection = ?", time.Now()).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update last connection for node %s: %w", id, err)
	}

	return &node, nil
}
