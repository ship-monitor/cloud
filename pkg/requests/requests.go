package requests

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var ErrNoParam = errors.New("no such param specified")

func GetParamUUID(c *gin.Context, key string) (uuid.UUID, error) {
	val, found := c.Params.Get(key)
	if !found {
		return uuid.Nil, fmt.Errorf("get param uuid %q: %w", key, ErrNoParam)
	}

	id, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed parse param %q as uuid: %w", val, err)
	}

	return id, nil
}

func MustGetParamUUID(c *gin.Context, key string) uuid.UUID {
	id, err := GetParamUUID(c, key)
	if err != nil {
		err = fmt.Errorf("bad %q specification: %w", key, err)
		c.AbortWithStatusJSON(http.StatusBadRequest, BadResponse{
			Details: err.Error(),
		})

		panic(err)
	}

	return id
}

type BadResponse struct {
	Details         string `json:"details"`
	MessageForArmen string `json:"messageForArmen,omitempty"`
}

func ResponseErr(err error) BadResponse {
	return BadResponse{
		Details:         err.Error(),
		MessageForArmen: "",
	}
}

func ResponseBad(details string) BadResponse {
	return BadResponse{
		Details:         details,
		MessageForArmen: "",
	}
}

func ResponseArmenErr(err error, msg string) BadResponse {
	return BadResponse{
		Details:         err.Error(),
		MessageForArmen: msg,
	}
}
