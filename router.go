package main

import (
	"Xiaohongshu_Simulator/models"
	"Xiaohongshu_Simulator/views"
	"github.com/gin-gonic/gin"
	"net/http"
)

func initRoutes(r *gin.Engine) {
	// 首页
	r.GET("", func(c *gin.Context) {
		// 获取帖子列表
		var Posts []models.Post
		if err := models.DB.Preload("User").Where("visible = ? and deleted = ?", true, false).Order("created_at desc").Find(&Posts).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取数据失败"})
			return
		}

		//是否登录逻辑的判断(这个需要删吗，有下面的逻辑的话)
		needsLogin := false
		loggedInUserID := ""
		cookieID, err := c.Cookie("user_id")
		if err != nil {
			needsLogin = true
		} else {
			loggedInUserID = cookieID
		}
		// 不报错不管就完事了
		var currentUser models.User
		if cookieID, err := c.Cookie("user_id"); err == nil {
			needsLogin = false
			loggedInUserID = cookieID
			models.DB.First(&currentUser, loggedInUserID)
		} else {
			needsLogin = true
		}

		for i := range Posts {
			var count int
			models.DB.Model(&models.Like{}).Where("post_id = ?", Posts[i].ID).Count(&count)
			Posts[i].LikeCount = count

			// 判断登录了的用户有没有点赞
			if currentUser.ID != 0 {
				var like models.Like
				if err := models.DB.Where("user_id = ? AND post_id = ?", currentUser.ID, Posts[i].ID).First(&like).Error; err == nil {
					Posts[i].IsLiked = true
				}
			}
		}

		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"Posts":          Posts,
			"NeedsLogin":     needsLogin,
			"LoggedInUserID": loggedInUserID,
		})
	})

	// 发布页面路由
	r.GET("/publish", func(c *gin.Context) {
		loggedInUserID := ""

		// 登录验证
		cookieID, err := c.Cookie("user_id")
		if err != nil { // 未登录则返回主页登录
			c.Redirect(http.StatusFound, "/")
			return
		}
		loggedInUserID = cookieID

		c.HTML(http.StatusOK, "publish.tmpl", gin.H{"LoggedInUserID": loggedInUserID})
	})

	api := r.Group("/api")
	{
		api.POST("/register", views.UserRegister)
		api.POST("/login", views.UserLogin)
		api.POST("/logout", views.UserLogout)
		api.POST("/user/update", views.UpdateUserProfile)
		api.POST("/post/create", views.CreatePost)
		api.GET("/post/detail", views.GetPost)
		api.POST("/post/like", views.ToggleLike)
		api.POST("/post/collect", views.ToggleCollect)
		api.POST("/post/comment", views.CreateComment)
	}

	user := r.Group("/user")
	{
		user.GET("/profile/:id", func(c *gin.Context) {
			targetID := c.Param("id")

			var User models.User
			var Posts []models.Post

			if err := models.DB.Where("id = ?", c.Param("id")).First(&User).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "获取用户id失败",
				})
				return
			}
			models.DB.Where("user_id = ? and deleted = ?", User.ID, false).Order("created_at desc").Find(&Posts)

			loggedInUserID := ""
			cookieID, err := c.Cookie("user_id")
			if err == nil {
				loggedInUserID = cookieID
			}

			isOwner := loggedInUserID == targetID

			var currentUser models.User
			if cookieID, err := c.Cookie("user_id"); err == nil {
				loggedInUserID = cookieID
				models.DB.First(&currentUser, loggedInUserID)
			}

			for i := range Posts {
				var count int
				models.DB.Model(&Posts[i]).Count(&count)
				Posts[i].LikeCount = count

				// 判断登录了的用户有没有点赞
				if currentUser.ID != 0 {
					var like models.Like
					if err := models.DB.Where("user_id = ? AND post_id = ?", currentUser.ID, Posts[i].ID).First(&like).Error; err == nil {
						Posts[i].IsLiked = true
					}
				}
			}

			c.HTML(http.StatusOK, "profile.tmpl", gin.H{
				"User":           User,
				"Posts":          Posts,
				"IsOwner":        isOwner,
				"LoggedInUserID": loggedInUserID,
			})
		})
	}
}
