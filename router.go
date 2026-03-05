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
		var Posts []models.Post

		if err := models.DB.Where("visible = ? and deleted = ?", true, false).Find(&Posts).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "获取数据失败",
			})
			return
		}
		//模拟数据

		post1 := models.Post{
			Title:   "dwadw",
			Text:    "dwadwafrggrdth",
			Visible: true,
			Deleted: false,
		}

		post2 := models.Post{
			Title:   "lijl",
			Text:    "jytrurt",
			Visible: true,
			Deleted: false,
		}

		Posts = append(Posts, post1, post2)

		needsLogin := false
		loggedInUserID := ""
		cookieID, err := c.Cookie("user_id")
		if err != nil {
			needsLogin = true
		} else {
			loggedInUserID = cookieID
		}

		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"Posts":          Posts,
			"NeedsLogin":     needsLogin,
			"LoggedInUserID": loggedInUserID,
		})
	})

	api := r.Group("/api")
	{
		api.POST("/register", views.UserRegister)
		api.POST("/login", views.UserLogin)
		api.POST("/user/update", views.UpdateUserProfile)
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

			c.HTML(http.StatusOK, "profile.tmpl", gin.H{
				"User":           User,
				"Posts":          Posts,
				"IsOwner":        isOwner,
				"LoggedInUserID": loggedInUserID,
			})
		})
	}
}
