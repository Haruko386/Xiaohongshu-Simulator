package views

import (
	"Xiaohongshu_Simulator/models"
	"github.com/gin-gonic/gin"
	"net/http"
)

type commentReq struct {
	PostID uint   `json:"post_id"`
	Text   string `json:"text"`
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
		Text:   req.Text,
		PostID: req.PostID,
		Post:   post,
		User:   user,
		UserID: user.ID,
	}

	if err := models.DB.Create(&newComment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评论失败，数据库错误"})
		return
	}

	// 评论成功
	c.JSON(http.StatusOK, gin.H{"message": "评论成功"})
}
