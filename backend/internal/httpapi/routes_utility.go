package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"novel-agent-runtime/internal/logging"
	"novel-agent-runtime/internal/workflow"
)

func registerUtilityRoutes(router *gin.Engine, deps routeDeps) {
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	router.GET("/v1/skills", func(c *gin.Context) {
		provider := workflow.LocalSkillProvider{SkillsDir: deps.cfg.Runtime.SkillsDir}
		skills, err := provider.ListSkills(c.Request.Context())
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		logging.FromContext(c.Request.Context()).Info("skill.provider.list", zap.Int("count", len(skills)))
		c.JSON(http.StatusOK, gin.H{"skills": skills})
	})
}
