package views

import (
	"Xiaohongshu_Simulator/models"
	"github.com/gin-gonic/gin"
	"net/http"
)

type commentReq struct {
	PostID   uint   `json:"post_id"`
	Text     string `json:"text"`
	ParentID uint   `json:"parent_id"`
	ReplyTo  string `json:"reple_to"`
}

func CreateComment(c *gin.Context) {
	// 先判断登录
	userStrID, err := c.Cookie("user_id")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未登录"})
		return
	}

	// 获取传来的评论请求
	var req commentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 获取到了评论的帖子的ID和当前用户的ID，我们获取相关的数据项
	var user models.User
	if err := models.DB.First(&user, userStrID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到用户"})
		return
	}

	var post models.Post
	if err := models.DB.First(&post, req.PostID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到帖子"})
		return
	}

	// 准备创建数据项并保存
	newComment := models.Comment{
		Text:     req.Text,
		PostID:   req.PostID,
		Post:     post,
		User:     user,
		UserID:   user.ID,
		ParentID: req.ParentID,
		ReplyTo:  req.ReplyTo,
	}

	if err := models.DB.Create(&newComment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评论失败，数据库错误"})
		return
	}

	// 评论成功
	c.JSON(http.StatusOK, gin.H{"message": "评论成功"})
}

type CommentLikeReq struct {
	CommentID uint `json:"comment_id"`
}

func ToggleCommentLike(c *gin.Context) {
	// 登录验证
	userStrID, err := c.Cookie("user_id")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var req CommentLikeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	var user models.User
	if err := models.DB.First(&user, userStrID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return
	}

	var commentLike models.CommentLike
	isLiked := false

	err = models.DB.Where("user_id = ? AND comment_id = ?", user.ID, req.CommentID).First(&commentLike).Error
	if err == nil {
		models.DB.Unscoped().Delete(&commentLike)
		isLiked = false
	} else {
		newCommentLike := models.CommentLike{
			UserID:    user.ID,
			CommentID: req.CommentID,
		}
		models.DB.Create(&newCommentLike)
		isLiked = true
	}

	var likeCount int
	models.DB.Model(&models.CommentLike{}).Where("comment_id = ?", req.CommentID).Count(&likeCount)

	c.JSON(http.StatusOK, gin.H{
		"is_liked":   isLiked,
		"like_count": likeCount,
	})
}
