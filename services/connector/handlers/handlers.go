package handlers

import (
	"net/http"
	"time"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
)

type ResponseNode struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	LastConnection  *time.Time `json:"lastConnection"`
	FirstConnection time.Time  `json:"firstConnection"`
}

func toResponse(in *domain.Node) ResponseNode {
	return ResponseNode{
		ID:              in.ID,
		Name:            in.Name,
		LastConnection:  in.LastConnection,
		FirstConnection: in.FirstConnection,
	}
}

type Node struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Handlers struct {
	repo domain.NodesRepo
}

func NewHandlers(repo domain.NodesRepo) *Handlers {
	return &Handlers{
		repo: repo,
	}
}

func (h *Handlers) GetSingleClientHandler() gin.HandlerFunc {
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

		node, err := h.repo.GetNode(c.Request.Context(), id)
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
