package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerRunRoutes(router *gin.Engine, deps routeDeps) {
	service := NewRunService(deps.cfg, deps.debug, deps.db, deps.models, deps.projects)

	router.POST("/v1/runs", func(c *gin.Context) {
		var req runRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		resp, err := service.Execute(c.Request.Context(), req)
		if err != nil {
			writeHTTPError(c, errorStatus(err, http.StatusInternalServerError), err)
			return
		}
		c.JSON(http.StatusOK, resp)
	})
}
