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
