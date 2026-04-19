package category

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type CreateCategoryRequest struct {
	Name      string `json:"name" binding:"required"`
	Slug      string `json:"slug" binding:"required"`
	SortOrder int    `json:"sort_order"`
	Status    uint8  `json:"status"`
}

type UpdateCategoryRequest struct {
	Name      string `json:"name" binding:"required"`
	Slug      string `json:"slug" binding:"required"`
	SortOrder int    `json:"sort_order"`
	Status    uint8  `json:"status"`
}

// 前台：获取启用分类
func (h *Handler) List(c *gin.Context) {
	categories, err := h.service.ListEnabled(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "get categories failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": categories,
	})
}

// 后台：获取所有分类
func (h *Handler) AdminList(c *gin.Context) {
	withDeleted := c.Query("with_deleted") == "1"

	categories, err := h.service.ListAll(c.Request.Context(), withDeleted)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "get categories failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": categories,
	})
}

// 获取单个分类
func (h *Handler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid category id"})
		return
	}

	category, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		switch err {
		case ErrCategoryNotFound:
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": category,
	})
}

// 获取单个分类
func (h *Handler) GetByName(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid category name"})
		return
	}

	category, err := h.service.GetByName(c.Request.Context(), name)
	if err != nil {
		switch err {
		case ErrCategoryNotFound:
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": category,
	})
}

func (h *Handler) GetBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid category slug"})
		return
	}

	category, err := h.service.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		switch err {
		case ErrCategoryNotFound:
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": category,
	})
}

// 新增分类（管理员）
func (h *Handler) Create(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid request"})
		return
	}

	category, err := h.service.Create(c.Request.Context(), req.Name, req.Slug, req.SortOrder, req.Status)
	if err != nil {
		switch err {
		case ErrCategoryExists:
			c.JSON(http.StatusConflict, gin.H{"code": 409, "msg": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "category created",
		"data": category,
	})
}

// 修改分类（管理员）
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid category id"})
		return
	}

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid request"})
		return
	}

	category, err := h.service.Update(c.Request.Context(), id, req.Name, req.Slug, req.SortOrder, req.Status)
	if err != nil {
		switch err {
		case ErrCategoryNotFound:
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": err.Error()})
		case ErrCategoryExists:
			c.JSON(http.StatusConflict, gin.H{"code": 409, "msg": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "category updated",
		"data": category,
	})
}

// 删除分类（管理员）
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid category id"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		switch err {
		case ErrCategoryNotFound:
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "category deleted",
	})
}

// 恢复分类（管理员）
func (h *Handler) Restore(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid category id"})
		return
	}

	if err := h.service.Restore(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "category restored",
	})
}
