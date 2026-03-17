package views

import (
	"Xiaohongshu_Simulator/models"
	"Xiaohongshu_Simulator/socket"
	"Xiaohongshu_Simulator/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func CreatePost(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var user models.User
	if err := models.DB.First(&user, userID).Error; err != nil {
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

func EditPost(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	postID := c.Param("id")

	var post models.Post
	if err := models.DB.First(&post, postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}

	if post.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限"})
		return
	}

	// 获取基本文本数据
	newTitle := c.PostForm("title")
	newText := c.PostForm("text")

	// 将前端传来的字符串转换为布尔值
	isVisible := true
	if c.PostForm("visible") == "false" {
		isVisible = false
	}

	//将要更新的字段打包进 map
	updates := map[string]interface{}{
		"Title":   newTitle,
		"Text":    newText,
		"Visible": isVisible,
	}

	//处理可选的封面图片上传
	file, err := c.FormFile("cover")
	if err == nil { // 如果 err == nil，说明用户上传了新图片
		var user models.User
		models.DB.First(&user, userID)

		// 确保存储目录存在
		postDir := fmt.Sprintf("./assets/%s/Posts/%d", user.Username, post.ID)
		os.MkdirAll(postDir, 0755)

		// 真正将文件保存到硬盘
		savePath := filepath.Join(postDir, file.Filename)
		if err := c.SaveUploadedFile(file, savePath); err == nil {
			// 保存成功后，才更新数据库里的文件名 (注意数据库字段叫 CoverImage)
			updates["CoverImage"] = file.Filename
		}
	}

	// 一次性执行多字段更新
	if err := models.DB.Model(&post).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新数据库失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "修改成功"})
}

func DeletePost(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	postID := c.Param("id")

	var post models.Post
	if err := models.DB.First(&post, postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "要删除的帖子不存在"})
		return
	}

	if userID != post.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限"})
		return
	}

	if err := models.DB.Model(&post).Update("deleted", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})
}

func GetPost(c *gin.Context) {
	// 获取要访问的笔记的id
	postIDStr := c.Param("id")
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

	//获取评论列表
	var comments []models.Comment // 按时间顺序获取
	if err := models.DB.Preload("User").Where("post_id = ?", postIDStr).Order("created_at desc").Find(&comments).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取评论列表失败"})
		return
	}

	// 查询当前用户，用于后续验证是否点赞了
	var user models.User
	isLoginned := false
	if tokenStr, err := c.Cookie("token"); err == nil {
		if claims, err := utils.ParseToken(tokenStr); err == nil {
			if err := models.DB.First(&user, claims.UserID).Error; err == nil {
				isLoginned = true
			}
		}
	}

	for i := range comments {
		var count int
		models.DB.Model(&models.CommentLike{}).Where("comment_id = ?", comments[i].ID).Count(&count)
		comments[i].LikeCount = count

		if isLoginned {
			var commentLike models.CommentLike
			if err := models.DB.Model(models.CommentLike{}).Where("comment_id = ? AND user_id = ?", comments[i].ID, user.ID).First(&commentLike).Error; err == nil {
				comments[i].IsLiked = true
			}
		}
	}

	var likeCount, collectCount int
	models.DB.Model(&models.Like{}).Where("post_id = ?", postIDStr).Count(&likeCount)
	models.DB.Model(&models.Collection{}).Where("post_id = ?", postIDStr).Count(&collectCount)

	isLiked := false
	isCollected := false

	if isLoginned {
		var tempLike models.Like
		if models.DB.Where("user_id = ? AND post_id = ?", user.ID, post.ID).First(&tempLike).Error == nil {
			isLiked = true
		}

		var tempCollected models.Collection
		if models.DB.Where("user_id = ? AND post_id = ?", user.ID, post.ID).First(&tempCollected).Error == nil {
			isCollected = true
		}
	}

	isFollowing := false
	if isLoginned {
		if err := models.DB.Model(&models.Follow{}).Where("follower_id = ? AND followee_id = ?", user.ID, post.UserID).First(&models.Follow{}).Error; err == nil {
			isFollowing = true
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
		"comments":      comments,
		"is_following":  isFollowing,
	})
}

// ActionReq 实现点赞本身的功能
type ActionReq struct {
	PostID uint `json:"post_id"`
}

func ToggleLike(c *gin.Context) {
	// 这种事情永远先检查登录
	userID := c.MustGet("user_id").(uint)

	// RESTful API ?
	postIDStr := c.Param("id")
	postID, _ := strconv.Atoi(postIDStr)

	// 接收请求，从json来的数据则使用`ShouldBindJSON`
	//var req ActionReq
	//if err := c.ShouldBindJSON(&req); err != nil {
	//	c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
	//	return
	//}

	// 查询是否已经点赞
	var like models.Like
	isLiked := false

	// postID与userID同时存在则已经点过赞,现在需要取消点赞
	err := models.DB.Where("user_id = ? AND post_id = ?", userID, uint(postID)).First(&like).Error
	if err == nil {
		// 记录存在，则正面以前点过赞，现在需要取消点赞，则删掉
		models.DB.Unscoped().Delete(&like)
		isLiked = false
	} else {
		// 记录不存在，则以前未点赞，现在加入

		var user models.User
		models.DB.First(&user, userID)

		newLike := models.Like{
			UserID: user.ID,
			PostID: uint(postID),
		}
		models.DB.Create(&newLike)
		isLiked = true

		// 发送通知    先查询帖子是谁写的
		var post models.Post // 第二个逻辑用于不给自己发通知
		if err := models.DB.First(&post, postID).Error; err == nil && userID != post.UserID {
			newNotif := models.Notification{
				UserID:     post.UserID,
				FromUserID: userID,
				Type:       "like_post",
				TargetID:   uint(postID),
				Content:    "",
			}
			models.DB.Create(&newNotif)

			// 呼叫 WebSocket 基站，尝试给在线的帖子作者发推送
			socket.GlobalManager.SendMessage(post.UserID, gin.H{
				"type": "new_notification",
				"msg":  user.Username + "刚刚赞了你的帖子",
			})
		}
	}

	var likeCount int
	models.DB.Model(&models.Like{}).Where("post_id = ?", uint(postID)).Count(&likeCount)

	c.JSON(http.StatusOK, gin.H{
		"is_liked":   isLiked,
		"like_count": likeCount,
	})
}

func ToggleCollect(c *gin.Context) {
	// 判断登录
	userID := c.MustGet("user_id").(uint)

	// 接收前端数据，json
	//var req ActionReq
	//if err := c.ShouldBindJSON(&req); err != nil {
	//	c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
	//	return
	//}  退场，学习RESTful API中

	postIDStr := c.Param("id")
	postID, _ := strconv.Atoi(postIDStr)

	// 查询是否已经收藏过
	var collect models.Collection
	isCollect := false

	// 查询收藏库里是否同时存在帖子与用户的对应
	err := models.DB.Where("user_id = ? AND post_id = ?", userID, uint(postID)).First(&collect).Error
	if err == nil {
		models.DB.Unscoped().Delete(&collect)
		isCollect = false
	} else {
		var user models.User
		models.DB.First(&user, userID)

		newCollect := models.Collection{
			UserID: user.ID,
			PostID: uint(postID),
		}
		models.DB.Create(&newCollect)
		isCollect = true

		var post models.Post
		if err := models.DB.First(&post, postID).Error; err == nil && userID != post.UserID {
			newNotif := models.Notification{
				UserID:     post.UserID,
				FromUserID: userID,
				Type:       "collect_post",
				TargetID:   uint(postID),
				Content:    "",
			}
			models.DB.Create(&newNotif)
			// 发送通知
			socket.GlobalManager.SendMessage(post.UserID, gin.H{
				"type": "new_notification",
				"msg":  user.Username + "收藏了你的帖子",
			})
		}
	}

	var collectCount int
	models.DB.Model(&models.Collection{}).Where("post_id = ?", uint(postID)).Count(&collectCount)

	c.JSON(http.StatusOK, gin.H{
		"is_collected":  isCollect,
		"collect_count": collectCount,
	})
}
