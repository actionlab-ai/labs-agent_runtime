package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"novel-agent-runtime/internal/config"
)

func registerWorkflowRoutes(router *gin.Engine, cfg config.Config, debug bool, db appStore, models modelConfigStore, projects projectConfigStore) {
	service := NewWorkflowService(cfg, debug, db, models, projects)

	router.POST("/v1/workflows/project-bootstrap", buildFixedWorkflowHandler(service, func(req workflowRunRequest) (fixedWorkflowPlan, error) {
		return buildProjectBootstrapPlan(req.Stage, req.Input, req.Arguments)
	}))
	router.POST("/v1/workflows/project-kickoff", buildFixedWorkflowHandler(service, func(req workflowRunRequest) (fixedWorkflowPlan, error) {
		return buildProjectKickoffPlan(req.Input, req.Arguments), nil
	}))
	router.POST("/v1/workflows/project-kernel", buildFixedWorkflowHandler(service, func(req workflowRunRequest) (fixedWorkflowPlan, error) {
		return buildProjectKernelPlan(req.Input, req.Arguments), nil
	}))
}

func buildFixedWorkflowHandler(service WorkflowService, planBuilder workflowPlanBuilder) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req workflowRunRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		resp, err := service.Execute(c.Request.Context(), req, planBuilder)
		if err != nil {
			writeHTTPError(c, errorStatus(err, http.StatusInternalServerError), err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}
