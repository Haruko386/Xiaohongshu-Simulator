package views

import (
	"Xiaohongshu_Simulator/models"
	"Xiaohongshu_Simulator/socket"
	"Xiaohongshu_Simulator/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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

	// 生成jwt token验证登录，不用cookie了
	token, err := utils.GenerateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "系统生成凭证失败"})
		return
	}

	// 这里是判断是否登录的，可以用cookie，也可以用JWT Token
	c.SetCookie("token", token, 3600*24, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"user_id": user.ID,
		"message": "登录成功",
	})
}

func UserLogout(c *gin.Context) {
	// 清空cookie
	c.SetCookie("token", "", -1, "/", "", false, true)
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
	userID := c.MustGet("user_id").(uint)
	var user models.User
	models.DB.Where("user_id = ?", userID).First(&user)

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
	userID := c.MustGet("user_id").(uint)

	targetIDStr := c.Param("id")
	targetID, err := strconv.Atoi(targetIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式错误"})
		return
	}

	var targetUser models.User
	if err := models.DB.First(&targetUser, targetID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "目标用户不存在"})
		return
	}

	//获取用户
	var user models.User
	if err := models.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
	}

	var Follow models.Follow
	isFollowing := false

	if err := models.DB.Where("follower_id = ? AND followee_id = ?", user.ID, targetUser.ID).First(&Follow).Error; err != nil {
		// 不空，则不存在关注A 2 B的关注，新建
		newFollow := models.Follow{
			FollowerID: user.ID,
			FolloweeID: targetUser.ID,
		}
		models.DB.Create(&newFollow)
		isFollowing = true

		newNotif := models.Notification{
			UserID:     targetUser.ID,
			FromUserID: userID,
			Type:       "follow",
			TargetID:   targetUser.ID,
			Content:    "",
		}
		models.DB.Create(&newNotif)

		socket.GlobalManager.SendMessage(targetUser.ID, gin.H{
			"type": "follow",
			"msg":  user.Username + "关注了你",
		})
	} else {
		// 存在A 2 B的关注，取关就进行删除
		models.DB.Unscoped().Delete(&Follow)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "操作成功",
		"is_following": isFollowing,
	})
}

func GetFollowingList(c *gin.Context) {
	userID := c.Param("id")
	var Follows []models.Follow

	if err := models.DB.Where("follower_id = ?", userID).Find(&Follows).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "查询错误"})
		return
	}

	// 获取所有关注的人的ID
	var followeeIDs []uint
	for _, f := range Follows {
		followeeIDs = append(followeeIDs, f.FolloweeID)
	}

	var users []models.User
	if len(followeeIDs) > 0 {
		models.DB.Where("id IN (?)", followeeIDs).Find(&users)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取列表成功",
		"users":   users,
	})
}

func GetFollowersList(c *gin.Context) {
	userID := c.Param("id")
	var Followers []models.Follow

	if err := models.DB.Model(&models.Follow{}).Where("followee_id = ?", userID).Find(&Followers).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "查询错误"})
		return
	}

	var followerIDs []uint
	for _, f := range Followers {
		followerIDs = append(followerIDs, f.FollowerID)
	}

	var users []models.User
	if len(followerIDs) > 0 {
		models.DB.Where("id in (?)", followerIDs).Find(&users)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取列表成功",
		"users":   users,
	})
}

func GetPostList(c *gin.Context) {
	types, userID := c.Query("type"), c.Query("user_id")
	var Posts []models.Post

	// 获取当前登录的用户
	var currentUser models.User
	isLoggedIn := false
	if tokenStr, err := c.Cookie("token"); err == nil {
		if claims, err := utils.ParseToken(tokenStr); err == nil {
			if err := models.DB.First(&currentUser, claims.UserID).Error; err == nil {
				isLoggedIn = true
			}
		}
	}

	if types == "created" {
		if err := models.DB.Preload("User").Where("visible = ? AND deleted = ? AND user_id = ?", true, false, userID).Order("created_at desc").Find(&Posts).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取数据失败"})
			return
		}
	} else if types == "collected" {
		var collections []models.Collection
		if err := models.DB.Where("user_id = ?", userID).Find(&collections).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取数据失败"})
			return
		}

		var postIDs []uint
		for _, p := range collections {
			postIDs = append(postIDs, p.PostID)
		}

		if len(postIDs) > 0 {
			models.DB.Preload("User").Where("id IN (?)", postIDs).Order("created_at desc").Find(&Posts)
		}
	} else if types == "liked" {
		var likes []models.Like
		if err := models.DB.Where("user_id = ?", userID).Find(&likes).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取数据失败"})
			return
		}

		var postIDs []uint
		for _, p := range likes {
			postIDs = append(postIDs, p.PostID)
		}

		if len(postIDs) > 0 {
			models.DB.Preload("User").Where("id IN (?)", postIDs).Order("created_at desc").Find(&Posts)
		}
	}

	// 点赞量之前没写
	for i := range Posts {
		// 点赞量
		var count int
		models.DB.Model(&models.Like{}).Where("post_id = ?", Posts[i].ID).Count(&count)
		Posts[i].LikeCount = count

		if isLoggedIn {
			var like models.Like
			if err := models.DB.Where("post_id = ? AND user_id = ?", Posts[i].ID, currentUser.ID).First(&like).Error; err == nil {
				Posts[i].IsLiked = true
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "查询成功",
		"posts":   Posts,
	})
}

func GetNotifications(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var Notifications []models.Notification // Preload 预查询表中其他表的数据
	if err := models.DB.Preload("FromUser").Where("user_id = ?", userID).Order("created_at desc").Find(&Notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取通知失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": Notifications,
	})
}

// MarkNotificationRead 标记全部消息已读
func MarkNotificationRead(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	if err := models.DB.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Update("is_read", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "标记已读失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "全部标记为已读"})
}
