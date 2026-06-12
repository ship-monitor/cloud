package handlers

import (
	"net/http"
	"time"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector/connections"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector/repository"
)

type ResponseNode struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	LastConnection  *time.Time `json:"lastConnection"`
	FirstConnection time.Time  `json:"firstConnection"`
	Connected       bool       `json:"connected"`
}

func toResponse(in *repository.Node) ResponseNode {
	connected := connections.IsConnected(in.ID)
	return ResponseNode{
		ID:              in.ID,
		Name:            in.Name,
		LastConnection:  in.LastConnection,
		FirstConnection: in.FirstConnection,
		Connected:       connected,
	}
}

type Node struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func GetSingleClientHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var uri struct {
			Id string `uri:"id" binding:"required"`
		}

		if err := c.BindUri(&uri); err != nil {
			log.Error("Failed bind uri", "error", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		id, err := uuid.Parse(uri.Id)
		if err != nil {
			log.Error("Failed parse id", "error", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		node, err := repository.GetNode(id)
		if err != nil {
			log.Error("Failed get nodes from repository", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"node": toResponse(node),
		})
	}
}
