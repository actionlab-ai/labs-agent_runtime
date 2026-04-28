package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"novel-agent-runtime/internal/store"
)

func registerProjectRoutes(router *gin.Engine, deps routeDeps) {
	router.GET("/v1/projects", func(c *gin.Context) {
		limit, offset := pagination(c)
		items, err := deps.projects.ListProjects(c.Request.Context(), limit, offset)
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"projects": items})
	})

	router.POST("/v1/projects", func(c *gin.Context) {
		var req projectCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		p, err := deps.projects.CreateProject(c.Request.Context(), store.CreateProjectParams{
			ID:              req.ID,
			Name:            req.Name,
			Description:     req.Description,
			StorageProvider: req.StorageProvider,
			StorageBucket:   req.StorageBucket,
			StoragePrefix:   req.StoragePrefix,
			Metadata:        req.Metadata,
		})
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"project": p})
	})

	router.GET("/v1/projects/:id", func(c *gin.Context) {
		p, err := deps.projects.GetProject(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeHTTPError(c, http.StatusNotFound, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"project": p})
	})

	router.PATCH("/v1/projects/:id", func(c *gin.Context) {
		var req projectUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		p, err := deps.projects.UpdateProject(c.Request.Context(), store.UpdateProjectParams{
			ID:              c.Param("id"),
			Name:            req.Name,
			Description:     req.Description,
			Status:          req.Status,
			StorageProvider: req.StorageProvider,
			StorageBucket:   req.StorageBucket,
			StoragePrefix:   req.StoragePrefix,
			Metadata:        req.Metadata,
		})
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"project": p})
	})

	router.DELETE("/v1/projects/:id", func(c *gin.Context) {
		if err := deps.projects.DeleteProject(c.Request.Context(), c.Param("id")); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	})
}

func registerProjectDocumentRoutes(router *gin.Engine, deps routeDeps) {
	router.GET("/v1/projects/:id/documents", func(c *gin.Context) {
		docs, err := deps.db.ListProjectDocuments(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeHTTPError(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"documents": docs})
	})

	router.PUT("/v1/projects/:id/documents/:kind", func(c *gin.Context) {
		var req projectDocumentUpsertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		kind := firstNonEmpty(req.Kind, c.Param("kind"))
		doc, err := deps.projects.UpsertProjectDocument(c.Request.Context(), store.UpsertProjectDocumentParams{
			ProjectID: c.Param("id"),
			Kind:      kind,
			Title:     firstNonEmpty(req.Title, kind),
			Body:      req.Body,
			Metadata:  req.Metadata,
		})
		if err != nil {
			writeHTTPError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"document": doc})
	})
}
