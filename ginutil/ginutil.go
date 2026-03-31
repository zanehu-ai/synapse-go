package ginutil

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/techfitmaster/synapse-go/resp"
)

// ParseIDParam reads ":id" from the route, writes a 400 error and returns false if invalid.
func ParseIDParam(c *gin.Context) (int64, bool) {
	return ParseParam(c, "id")
}

// ParseParam reads a named route param, writes a 400 error and returns false if not a valid int64.
func ParseParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, resp.CodeBadRequest, "invalid "+name)
		return 0, false
	}
	return id, true
}
