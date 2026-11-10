package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	"github.com/skryfon-org/employee-360/backend/internal/types/pagination"
)

// intQuery parses an integer query parameter; a missing or empty value yields def.
func intQuery(c *gin.Context, key string, def int) (int, error) {
	raw := c.Query(key)
	if raw == "" {
		return def, nil
	}
	return strconv.Atoi(raw)
}

// pageParams parses the shared page/page_size query params. Non-integer values
// write a 400 and return ok=false; out-of-range integers are passed through and
// clamped by pagination.Normalize in the usecase.
func pageParams(c *gin.Context) (page, pageSize int, ok bool) {
	page, err := intQuery(c, "page", pagination.DefaultPage)
	if err != nil {
		response.BadRequest(c, "page must be an integer")
		return 0, 0, false
	}
	pageSize, err = intQuery(c, "page_size", pagination.DefaultPageSize)
	if err != nil {
		response.BadRequest(c, "page_size must be an integer")
		return 0, 0, false
	}
	return page, pageSize, true
}

// pageMeta builds the uniform pagination meta from a usecase result.
func pageMeta(page, pageSize int, total int64) response.Meta {
	return response.Meta{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: pagination.TotalPages(total, pageSize),
	}
}
