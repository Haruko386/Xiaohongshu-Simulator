package main

import (
	"Xiaohongshu_Simulator/middleware"
	"Xiaohongshu_Simulator/models"
	"Xiaohongshu_Simulator/utils"
	"Xiaohongshu_Simulator/views"
	"fmt"
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
		needsLogin := true
		loggedInUserID := ""
		// 不报错不管就完事了
		var currentUser models.User

		if tokenStr, err := c.Cookie("token"); err == nil {
			if claims, err := utils.ParseToken(tokenStr); err == nil {
				needsLogin = false
				loggedInUserID = fmt.Sprint(claims.UserID)
				models.DB.First(&currentUser, claims.UserID)
			}
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
		tokenStr, err := c.Cookie("token")
		if err == nil {
			if claims, err := utils.ParseToken(tokenStr); err == nil {
				loggedInUserID = fmt.Sprint(claims.UserID)
			}
		}

		if loggedInUserID == "" {
			c.Redirect(http.StatusFound, "/")
			return
		}

		c.HTML(http.StatusOK, "publish.tmpl", gin.H{"LoggedInUserID": loggedInUserID})
	})

	api := r.Group("/api")
	{
		// 公共接口
		api.POST("/auth/register", views.UserRegister)
		api.POST("/auth/login", views.UserLogin)

		api.GET("/users/:id/following", views.GetFollowingList)
		api.GET("/users/:id/followers", views.GetFollowersList)

		api.GET("/posts/:id", views.GetPost)
		api.GET("/posts", views.GetPostList)

		// 需验证的接口
		authApi := api.Group("")
		authApi.Use(middleware.AuthRequired()) // 挂载AuthRequired中间件
		{
			authApi.POST("/auth/logout", views.UserLogout)

			authApi.PUT("/users/me", views.UpdateUserProfile)

			authApi.POST("/users/:id/follow", views.ToggleFollow)

			authApi.POST("/comments/:id/like", views.ToggleCommentLike)
			authApi.POST("/posts", views.CreatePost)
			authApi.POST("/posts/:id/like", views.ToggleLike)
			authApi.POST("/posts/:id/collect", views.ToggleCollect)
			authApi.POST("/posts/:id/comments", views.CreateComment)
		}

	}

	user := r.Group("/user")
	{
		user.GET("/profile/:id", func(c *gin.Context) {
			targetID := c.Param("id") // 这里的targetID是登录访问的个人页面的用户ID

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
			var currentUser models.User
			if tokenStr, err := c.Cookie("token"); err == nil {
				if claims, err := utils.ParseToken(tokenStr); err == nil {
					loggedInUserID = fmt.Sprint(claims.UserID)
					models.DB.First(&currentUser, claims.UserID)
				}
			}

			isOwner := (loggedInUserID != "") && (loggedInUserID == targetID)

			isFollowing := false
			if currentUser.ID != 0 { // 只有登录了才判断是否关注
				if err := models.DB.Where("follower_id = ? AND followee_id = ?", currentUser.ID, targetID).First(&models.Follow{}).Error; err == nil {
					isFollowing = true
				}
			}

			totalFavorite := 0

			for i := range Posts {
				var count, collect int
				models.DB.Model(models.Like{}).Where("post_id = ?", Posts[i].ID).Count(&count)
				Posts[i].LikeCount = count

				models.DB.Model(models.Collection{}).Where("post_id = ?", Posts[i].ID).Count(&collect)

				totalFavorite += count + collect

				// 判断登录了的用户有没有点赞
				if currentUser.ID != 0 {
					var like models.Like
					if err := models.DB.Where("user_id = ? AND post_id = ?", currentUser.ID, Posts[i].ID).First(&like).Error; err == nil {
						Posts[i].IsLiked = true
					}
				}
			}

			var followerCount, followingCount int

			models.DB.Model(&models.Follow{}).Where("followee_id = ?", targetID).Count(&followerCount)
			models.DB.Model(&models.Follow{}).Where("follower_id = ?", targetID).Count(&followingCount)

			c.HTML(http.StatusOK, "profile.tmpl", gin.H{
				"User":           User,
				"Posts":          Posts,
				"IsOwner":        isOwner,
				"LoggedInUserID": loggedInUserID,
				"FollowerCount":  followerCount,
				"FollowingCount": followingCount,
				"TotalFavorited": totalFavorite,
				"IsFollowing":    isFollowing,
			})
		})
	}
}
