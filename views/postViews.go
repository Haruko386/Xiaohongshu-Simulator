package views

import (
	"Xiaohongshu_Simulator/models"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func CreatePost(c *gin.Context) {
	userIDStr, err := c.Cookie("user_id")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"erroe": "请先登录"})
		return
	}

	var user models.User
	if err := models.DB.First(&user, userIDStr).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"erroe": "用户不存在"})
		return
	}

	title := c.PostForm("title")
	text := c.PostForm("text")
	file, err := c.FormFile("cover")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erroe": "封面为必选项"})
		return
	}

	now := time.Now()
	newPost := models.Post{
		Title:      title,
		Text:       text,
		UserID:     user.ID,
		Visible:    true,
		Deleted:    false,
		PublicDate: &now,
		EditDate:   &now,
	}

	if err := models.DB.Create(&newPost).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erroe": "创建笔记失败"})
		return
	}

	// 笔记存储路径
	postDir := fmt.Sprintf("./assets/%s/Posts/%d", user.Username, newPost.ID)
	if err := os.MkdirAll(postDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建笔记失败"})
		return
	}

	// 保存图像
	savePath := filepath.Join(postDir, file.Filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存封面失败"})
		return
	}

	models.DB.Model(&newPost).Update("CoverImage", file.Filename)

	c.JSON(http.StatusOK, gin.H{"message": "发布成功"})
}

func GetPost(c *gin.Context) {
	// 获取要访问的笔记的id
	postIDStr := c.Query("id")
	if postIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取的笔记无效"})
		return
	}
	// 获取到我们要查询的笔记
	var post models.Post
	if err := models.DB.Preload("User").First(&post, postIDStr).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在于数据库中"})
		return
	}

	var likeCount, collectCount int
	models.DB.Model(&models.Like{}).Where("post_id = ?", postIDStr).Count(&likeCount)
	models.DB.Model(&models.Collection{}).Where("post_id = ?", postIDStr).Count(&collectCount)

	isLiked := false
	isCollected := false

	userIDStr, err := c.Cookie("user_id")
	if err == nil {
		var user models.User
		models.DB.First(&user, userIDStr)

		var tempLike models.Like
		if models.DB.Where("user_id = ? AND post_id = ?", user.ID, post.ID).First(&tempLike).Error == nil {
			isLiked = true
		}

		var tempCollected models.Collection
		if models.DB.Where("user_id = ? AND post_id = ?", user.ID, post.ID).First(&tempCollected).Error == nil {
			isCollected = true
		}
	}

	// 返回呗？
	c.JSON(http.StatusOK, gin.H{
		"post":          post,
		"user":          post.User,
		"like_count":    likeCount,
		"collect_count": collectCount,
		"is_liked":      isLiked,
		"is_collected":  isCollected,
	})
}

// ActionReq 实现点赞本身的功能
type ActionReq struct {
	PostID uint `json:"post_id"`
}

func ToggleLike(c *gin.Context) {
	// 这种事情永远先检查登录
	userIDStr, err := c.Cookie("user_id")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未登录"})
		return
	}

	// 接收请求，从json来的数据则使用`ShouldBindJSON`
	var req ActionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 查询是否已经点赞
	var like models.Like
	isLiked := false

	// postID与userID同时存在则已经点过赞,现在需要取消点赞
	err = models.DB.Where("user_id = ? AND post_id = ?", userIDStr, req.PostID).First(&like).Error
	if err == nil {
		// 记录存在，则正面以前点过赞，现在需要取消点赞，则删掉
		models.DB.Delete(&like).Unscoped()
		isLiked = false
	} else {
		// 记录不存在，则以前未点赞，现在加入

		var user models.User
		models.DB.First(&user, userIDStr)

		newLike := models.Like{
			UserID: user.ID,
			PostID: req.PostID,
		}
		models.DB.Create(&newLike)
		isLiked = true
	}

	var likeCount int
	models.DB.Model(&models.Like{}).Where("post_id = ?", req.PostID).Count(&likeCount)

	c.JSON(http.StatusOK, gin.H{
		"is_liked":   isLiked,
		"like_count": likeCount,
	})
}

func ToggleCollect(c *gin.Context) {
	// 判断登录
	userIDStr, err := c.Cookie("user_id")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未登录"})
		return
	}

	// 接收前端数据，json
	var req ActionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 查询是否已经收藏过
	var collect models.Collection
	isCollect := false

	// 查询收藏库里是否同时存在帖子与用户的对应
	err = models.DB.Where("user_id = ? AND post_id = ?", userIDStr, req.PostID).First(&collect).Error
	if err == nil {
		models.DB.Delete(&collect).Unscoped()
		isCollect = false
	} else {
		var user models.User
		models.DB.First(&user, userIDStr)

		newCollect := models.Collection{
			UserID: user.ID,
			PostID: req.PostID,
		}
		models.DB.Create(&newCollect)
		isCollect = true
	}

	var collectCount int
	models.DB.Model(&models.Collection{}).Where("post_id = ?", req.PostID).Count(&collectCount)

	c.JSON(http.StatusOK, gin.H{
		"is_collect":    isCollect,
		"collect_count": collectCount,
	})
}
