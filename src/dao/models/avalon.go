package models

import "time"

type User struct {
	ID         int                   `gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY;"`
	Username   string                `gorm:"column:username;NOT NULL;"`
	Hash       string                `gorm:"column:hash;NOT NULL;"`
	Email      string                `gorm:"column:email;NOT NULL;"`
	QQ         int64                 `gorm:"column:qq;NOT NULL;"`
	Bilibili   []BilibiliEpisodeInfo `gorm:"many2many:subscription_bilibili;"`
	Mikan      []MikanAnimeSet       `gorm:"many2many:subscription_mikan;"`
	Permission int                   `gorm:"column:permission;NOT NULL;"`
	IsAdmin    bool                  `gorm:"column:is_admin;NOT NULL;"`
}

func (User) TableName() string {
	return "user"
}

type QQGroup struct {
	ID int64 `gorm:"column:id;NOT NULL;PRIMARY KEY;"`
	//Subsciptions []UserSubscription `gorm:"foreignKey:Id;references:bvid"`
	Permission int `gorm:"column:permission;NOT NULL;"`
}

func (QQGroup) TableName() string {
	return "qq_group"
}

type RuntimeControl struct {
	ID     int    `gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY;"`
	Action string `gorm:"column:action;NOT NULL;"`
	Data   string `gorm:"column:data;NOT NULL;"`
}

func (RuntimeControl) TableName() string {
	return "runtime_control"
}

type Announcements struct {
	ID   int    `gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY;"`
	Push bool   `gorm:"column:push;NOT NULL;"`
	Data string `gorm:"column:data;NOT NULL;"`
}

func (Announcements) TableName() string {
	return "announcements"
}

type Settings struct {
	ID                 int       `gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY;"`
	BilibiliLastUpdate time.Time `gorm:"column:bilibiliLastUpdate;NOT NULL;"`
}

func (Settings) TableName() string {
	return `settings`
}
