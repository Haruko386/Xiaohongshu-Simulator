package views

import (
	"Xiaohongshu_Simulator/models"
	"Xiaohongshu_Simulator/socket"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type commentReq struct {
	PostID   uint   `json:"post_id"`
	Text     string `json:"text"`
	ParentID uint   `json:"parent_id"`
	ReplyTo  string `json:"reply_to"`
}

func CreateComment(c *gin.Context) {
	// 先判断登录    新的中间件之后直接不用在这里判断了，中间件判断是否登录了
	userID := c.MustGet("user_id").(uint)

	// 获取传来的评论请求
	var req commentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 获取到了评论的帖子的ID和当前用户的ID，我们获取相关的数据项
	var user models.User
	if err := models.DB.First(&user, userID).Error; err != nil {
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

	notifyUserID := post.UserID // 默认发给楼主

	if req.ParentID != 0 {
		var parentComment models.Comment
		// 如果是楼中楼，查出被回复的那条评论是谁发的
		if err := models.DB.First(&parentComment, req.ParentID).Error; err == nil {
			notifyUserID = parentComment.UserID
		}
	}

	// 触发通知 (条件是：不要自己给自己发通知)
	if notifyUserID != userID {
		newNotif := models.Notification{
			UserID:     notifyUserID,
			FromUserID: userID,
			Type:       "comment",
			TargetID:   req.PostID,
			Content:    req.Text,
		}
		models.DB.Create(&newNotif)

		socket.GlobalManager.SendMessage(notifyUserID, gin.H{
			"type": "new_notification",
			"msg":  user.Username + " 回复了你",
		})
	}

	// 评论成功
	c.JSON(http.StatusOK, gin.H{"message": "评论成功"})
}

type CommentLikeReq struct {
	CommentID uint `json:"comment_id"`
}

func ToggleCommentLike(c *gin.Context) {
	// 登录验证
	userID := c.MustGet("user_id").(uint)

	var req CommentLikeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	commentIDStr := c.Param("id")
	commentID, _ := strconv.Atoi(commentIDStr)

	var user models.User
	if err := models.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return
	}

	var commentLike models.CommentLike
	isLiked := false

	err := models.DB.Where("user_id = ? AND comment_id = ?", user.ID, uint(commentID)).First(&commentLike).Error
	if err == nil {
		models.DB.Unscoped().Delete(&commentLike)
		isLiked = false
	} else {
		newCommentLike := models.CommentLike{
			UserID:    user.ID,
			CommentID: uint(commentID),
		}
		models.DB.Create(&newCommentLike)
		isLiked = true

		var comment models.Comment
		if err := models.DB.First(&comment, commentLike.CommentID).Error; err == nil && comment.UserID != userID {
			newNotif := models.Notification{
				UserID:     comment.UserID,
				FromUserID: userID,
				Type:       "like_comment",
				TargetID:   comment.ID,
				Content:    "",
			}
			models.DB.Create(&newNotif)

			socket.GlobalManager.SendMessage(comment.UserID, gin.H{
				"type": "new_notification",
				"msg":  user.Username + "点赞了你的评论",
			})
		}
	}

	var likeCount int
	models.DB.Model(&models.CommentLike{}).Where("comment_id = ?", uint(commentID)).Count(&likeCount)

	c.JSON(http.StatusOK, gin.H{
		"is_liked":   isLiked,
		"like_count": likeCount,
	})
}
