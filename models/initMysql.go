package models

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"time"
)

var DB *gorm.DB

func InitMysql() (err error) {
	dsn := "root:password@(127.0.0.1:3306)/xiaohongshu?charset=utf8mb4&parseTime=True&loc=Local"
	DB, err = gorm.Open("mysql", dsn)
	if err != nil {
		return err
	}
	err = DB.DB().Ping()
	return err
}

type User struct {
	gorm.Model
	Username  string    `gorm:"unique" json:"username"`
	Password  string    `gorm:"size:255" json:"password"`
	Avatar    string    `gorm:"size:255;" json:"avatar"`
	Signature string    `gorm:"size:255" json:"signature"`
	Gender    string    `gorm:"default:'male'" json:"gender"`
	Birthday  time.Time `gorm:"type:date" json:"birthday"`
	Region    string    `gorm:"type:varchar(255)" json:"region"`
}

type Post struct {
	gorm.Model
	Title      string    `gorm:"type:varchar(255)" json:"title"`
	Text       string    `gorm:"type:varchar(255)" json:"text"`
	CoverImage string    `gorm:"type:varchar(255)" json:"cover_image"`
	Visible    bool      `gorm:"type:bool;default:true" json:"visible"`
	PublicDate time.Time `gorm:"type:date" json:"public_date"`
	EditDate   time.Time `gorm:"type:date" json:"edit_date"`
	Deleted    bool      `gorm:"type:bool;default:false" json:"deleted"`

	UserID uint `json:"user_id"`
}
