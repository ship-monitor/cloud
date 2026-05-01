package repository

import (
	"context"
	"fmt"
	"time"

	"charm.land/log/v2"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
)

func Migrate() {
	ctx := context.TODO()
	query := db.DB.NewCreateTable().
		Model(&Node{}).
		IfNotExists()

	_, err := query.Exec(ctx)
	if err != nil {
		fmt.Println(query.String())
		log.Error("failed to create node table", "error", err.Error())
		panic(err)
	}
}

type Node struct {
	*bun.BaseModel  `bun:"table:nodes"`
	ID              uuid.UUID  `bun:",pk,type:varchar" json:"id"`
	Name            string     `bun:",notnull" json:"name"`
	FirstConnection time.Time  `bun:",notnull,type:varchar" json:"firstConnection"`
	LastConnection  *time.Time `bun:",notnull,type:varchar" json:"lastConnection"`
}

func GetNode(id uuid.UUID) (*Node, error) {
	var node Node
	err := db.DB.NewSelect().Model(&node).Where("id = ?", id).Scan(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed select node with id %s: %s", id, err)
	}
	return &node, nil
}
func GetNodes(organizationID uuid.UUID) ([]Node, error) {
	var nodes []Node
	err := db.DB.NewSelect().Model(&nodes).Where("organization_id = ?", organizationID).Scan(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes for organization %s: %s", organizationID, err)
	}
	return nodes, nil
}

func NewNode(id uuid.UUID, name string) (*Node, error) {
	now := time.Now()
	model := &Node{
		ID:              id,
		Name:            name,
		LastConnection:  &now,
		FirstConnection: now,
	}
	_, err := db.DB.NewInsert().Model(model).Exec(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %s", err)
	}
	return model, nil
}

func ReconnectNode(id uuid.UUID) (*Node, error) {
	var node Node
	_, err := db.DB.NewUpdate().Model(&node).Where("id = ?", id).Set("last_connection = ?", time.Now()).Exec(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to reconnect node %s: %s", id, err)
	}
	return &node, nil
}

func UpdateLastConnection(id uuid.UUID) (*Node, error) {
	var node Node
	_, err := db.DB.NewUpdate().Model(&node).Where("id = ?", id).Set("last_connection = ?", time.Now()).Exec(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to update last connection for node %s: %s", id, err)
	}
	return &node, nil
}
