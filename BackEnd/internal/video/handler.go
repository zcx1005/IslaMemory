package video

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler 负责 video HTTP 请求处理
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List 处理 GET /api/v1/videos
// 支持：分页、分类筛选、关键词搜索、排序
func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	input := ListInput{
		Page:         page,
		PageSize:     pageSize,
		CategorySlug: c.Query("category_slug"),
		Keyword:      c.Query("keyword"),
		Sort:         c.DefaultQuery("sort", "latest"),
	}

	list, total, err := h.service.ListPublicVideos(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "get videos failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": gin.H{
			"list":      list,
			"total":     total,
			"page":      input.Page,
			"page_size": input.PageSize,
		},
	})
}

// Detail 处理 GET /api/v1/videos/:public_id
func (h *Handler) Detail(c *gin.Context) {
	publicID := c.Param("public_id")
	if publicID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid public_id",
		})
		return
	}

	detail, err := h.service.GetPublicVideoDetail(c.Request.Context(), publicID)
	if err != nil {
		switch err {
		case ErrVideoNotFound:
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": detail,
	})
}

// TODO: IncreasePlay 处理 POST /api/v1/videos/:public_id/play
func (h *Handler) IncreasePlay(c *gin.Context) {
	publicID := c.Param("public_id")
	if publicID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid public_id",
		})
		return
	}

	if err := h.service.IncreasePlayCount(c.Request.Context(), publicID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "increase play count failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "play count increased",
	})
}
