package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"novel-agent-runtime/internal/logging"
	"novel-agent-runtime/internal/project"
	"novel-agent-runtime/internal/store"
)

func registerModelRoutes(router *gin.Engine, deps routeDeps) {
	router.GET("/v1/models", func(c *gin.Context) {
		limit, offset := pagination(c)
		items, err := deps.db.ListModelProfiles(c.Request.Context(), limit, offset)
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"models": items})
	})

	router.POST("/v1/models", func(c *gin.Context) {
		var req modelProfileCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		item, err := deps.db.CreateModelProfile(c.Request.Context(), store.CreateModelProfileParams{
			ID:              req.ID,
			Name:            req.Name,
			Provider:        req.Provider,
			ModelID:         req.ModelID,
			BaseURL:         req.BaseURL,
			APIKey:          req.APIKey,
			ContextWindow:   req.ContextWindow,
			MaxOutputTokens: req.MaxOutputTokens,
			Temperature:     req.Temperature,
			TimeoutSeconds:  req.TimeoutSeconds,
			Metadata:        req.Metadata,
		})
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		logging.FromContext(c.Request.Context()).Info("model.pg.create",
			zap.String("profile_id", item.ID),
			zap.String("model_id", item.ModelID),
			zap.String("provider", item.Provider),
		)
		deps.models.CacheModelProfile(c.Request.Context(), item)
		c.JSON(http.StatusOK, gin.H{"model": item})
	})

	router.GET("/v1/models/:id", func(c *gin.Context) {
		item, err := deps.models.GetModelProfile(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeHTTPError(c, http.StatusNotFound, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"model": item})
	})

	router.PATCH("/v1/models/:id", func(c *gin.Context) {
		var req modelProfileUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		item, err := deps.db.UpdateModelProfile(c.Request.Context(), store.UpdateModelProfileParams{
			ID:              c.Param("id"),
			Name:            req.Name,
			Provider:        req.Provider,
			ModelID:         req.ModelID,
			BaseURL:         req.BaseURL,
			APIKey:          req.APIKey,
			ContextWindow:   req.ContextWindow,
			MaxOutputTokens: req.MaxOutputTokens,
			Temperature:     req.Temperature,
			TimeoutSeconds:  req.TimeoutSeconds,
			Status:          req.Status,
			Metadata:        req.Metadata,
		})
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		logging.FromContext(c.Request.Context()).Info("model.pg.update",
			zap.String("profile_id", item.ID),
			zap.String("model_id", item.ModelID),
			zap.String("provider", item.Provider),
			zap.String("status", item.Status),
		)
		deps.models.CacheModelProfile(c.Request.Context(), item)
		c.JSON(http.StatusOK, gin.H{"model": item})
	})

	router.DELETE("/v1/models/:id", func(c *gin.Context) {
		defaultModelID, err := deps.db.GetDefaultModelID(c.Request.Context())
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		if defaultModelID != "" {
			modelProfile, err := deps.models.GetModelProfile(c.Request.Context(), c.Param("id"))
			if err != nil {
				writeHTTPError(c, http.StatusBadRequest, err)
				return
			}
			if defaultModelID == modelProfile.ID {
				writeHTTPError(c, http.StatusBadRequest, fmt.Errorf("model %q is the default model; change default first", c.Param("id")))
				return
			}
		}
		if err := deps.db.DeleteModelProfile(c.Request.Context(), c.Param("id")); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		logging.FromContext(c.Request.Context()).Info("model.pg.delete", zap.String("profile_id", project.Slug(c.Param("id"))))
		deps.models.DeleteModelProfile(c.Request.Context(), c.Param("id"))
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	})
}

func registerSettingRoutes(router *gin.Engine, deps routeDeps) {
	router.GET("/v1/settings/default-model", func(c *gin.Context) {
		modelID, err := deps.models.GetDefaultModelID(c.Request.Context())
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		if modelID == "" {
			c.JSON(http.StatusOK, gin.H{"default_model_id": "", "model": nil})
			return
		}
		modelProfile, err := deps.models.GetModelProfile(c.Request.Context(), modelID)
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"default_model_id": modelID, "model": modelProfile})
	})

	router.PUT("/v1/settings/default-model", func(c *gin.Context) {
		var req defaultModelUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		modelProfile, err := deps.models.GetModelProfile(c.Request.Context(), req.Model)
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		if modelProfile.Status != "" && modelProfile.Status != "active" {
			writeHTTPError(c, http.StatusBadRequest, fmt.Errorf("model profile %q is not active", modelProfile.ID))
			return
		}
		if err := deps.models.SetDefaultModelID(c.Request.Context(), modelProfile.ID); err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"default_model_id": modelProfile.ID, "model": modelProfile})
	})

	router.DELETE("/v1/settings/default-model", func(c *gin.Context) {
		if err := deps.models.ClearDefaultModelID(c.Request.Context()); err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	})

	router.GET("/v1/settings/project-document-policy", func(c *gin.Context) {
		setting, err := deps.db.GetAppSetting(c.Request.Context(), project.DocumentPolicySettingKey)
		if err != nil || setting.Value == "" {
			c.JSON(http.StatusOK, gin.H{"key": project.DocumentPolicySettingKey, "value": project.DefaultDocumentPolicy()})
			return
		}
		policy, err := project.ParseDocumentPolicy(setting.Value)
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"key": project.DocumentPolicySettingKey, "value": policy})
	})

	router.PUT("/v1/settings/project-document-policy", func(c *gin.Context) {
		var req projectDocumentPolicyUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		policy, err := project.ParseDocumentPolicy(string(req.Value))
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		body, err := json.Marshal(policy)
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		setting, err := deps.db.UpsertAppSetting(c.Request.Context(), project.DocumentPolicySettingKey, string(body))
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"key": setting.Key, "value": policy})
	})
}
