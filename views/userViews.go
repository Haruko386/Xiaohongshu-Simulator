package views

import (
	"Xiaohongshu_Simulator/models"
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type UserReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func UserRegister(c *gin.Context) {
	var req UserReq
	// 解析json
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 查看是否已存在用户
	var existUser models.User
	if err := models.DB.Where("username = ?", req.Username).First(&existUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "用户已存在"})
		return
	}

	hashPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加密失败"})
		return
	}

	newUser := models.User{
		Username: req.Username,
		Password: string(hashPassword),
		Avatar:   "default.png",
	}

	if err := models.DB.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败，数据库错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "注册成功"})
}

func UserLogin(c *gin.Context) {
	var req UserReq
	// 获取请求
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 去数据库找人
	var user models.User
	if err := models.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return
	}

	// 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	// 这里是判断是否登录的，可以用cookie，也可以用JWT Token
	c.SetCookie("user_id", fmt.Sprint(user.ID), 3600, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"user_id": user.ID,
		"message": "登录成功",
	})
}

func UserLogout(c *gin.Context) {
	// 清空cookie
	c.SetCookie("user_id", "", -1, "/", "", false, true)
	// 返回值
	c.JSON(http.StatusOK, gin.H{"message": "已退出登录"})
}

type UserUpdateReq struct {
	Signature string `json:"signature"`
	Gender    string `json:"gender"`
	Region    string `json:"region"`
}

func UpdateUserProfile(c *gin.Context) {
	// 验证登录状态
	userIDStr, err := c.Cookie("user_id")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var user models.User
	if err := models.DB.First(&user, userIDStr).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	gender := c.PostForm("gender")
	region := c.PostForm("region")
	signature := c.PostForm("signature")
	birthdayStr := c.PostForm("birthday")

	var birthday *time.Time
	if birthdayStr != "" {
		parsedTime, err := time.Parse("2006-01-02", birthdayStr)
		if err == nil {
			birthday = &parsedTime
		}
	}

	file, err := c.FormFile("avatar")
	if err == nil {
		userDir := fmt.Sprintf("./assets/%s", user.Username)

		if err := os.MkdirAll(userDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户文件夹失败"})
			return
		}
		// 保存路径
		savePath := filepath.Join(userDir, file.Filename)

		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "头像上传失败"})
			return
		}

		user.Avatar = file.Filename
	}

	user.Gender = gender
	user.Region = region
	user.Signature = signature
	user.Birthday = birthday

	if err := models.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户信息更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

type FollowReq struct {
	FollowerID uint `json:"target_id"`
}

func ToggleFollow(c *gin.Context) {
	// 获取当前登录的ID
	userStrID, err := c.Cookie("user_id")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未登录"})
		return
	}

	var req FollowReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	//获取用户
	var user models.User
	if err := models.DB.First(&user, userStrID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
	}

	var Follow models.Follow
	isFollowing := false

	if err := models.DB.Where("follower_id = ? AND followee_id = ?", user.ID, req.FollowerID).First(&Follow).Error; err != nil {
		// 不空，则不存在关注A 2 B的关注，新建
		newFollow := models.Follow{
			FollowerID: user.ID,
			FolloweeID: req.FollowerID,
		}
		models.DB.Create(&newFollow)
		isFollowing = true
	} else {
		// 存在A 2 B的关注，取关就进行删除
		models.DB.Unscoped().Delete(&Follow)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "操作成功",
		"is_following": isFollowing,
	})
}
