package dao

import (
	"avalon-core/src/dao/models"
	"gorm.io/gorm/clause"
)

func GetUserByQQ(qq int64) (*models.User, error) {
	db := GetMySQLInstance().Database

	var user models.User
	err := db.Preload(clause.Associations).Where(&models.User{QQ: qq}).First(&user).Error

	return &user, err
}

func CreateBilibiliSubscription(user *models.User, videoID string) error {
	db := GetMySQLInstance().Database

	episode, err := FindOrCreateBilibiliEpisodeBySeasonTag(videoID)
	if err != nil {
		return err
	}

	user.Bilibili = append(user.Bilibili, *episode)
	err = db.Preload(clause.Associations).Save(user).Error
	return err
}

func CreateNewMikanSubscription(user *models.User, animeID, subtitleGroupID int, bangumiTitle string) error {
	db := GetMySQLInstance().Database

	anime, err := FindOrCreateMikanAnimeSet(animeID, subtitleGroupID, bangumiTitle)
	if err != nil {
		return err
	}

	user.Mikan = append(user.Mikan, *anime)
	err = db.Preload(clause.Associations).Save(user).Error
	return err
}

func GetAdmin() ([]*models.User, error) {
	db := GetMySQLInstance().Database

	var admins []*models.User

	err := db.Preload(clause.Associations).Where(`is_admin = 1`).Find(&admins).Error
	return admins, err
}

func IsInAdmin(qq int64) bool {
	db := GetMySQLInstance().Database

	var count int64
	err := db.Model(&models.User{}).Where(`qq = ?`, qq).Where(`is_admin = 1`).Count(&count).Error
	if err != nil || count == 0 {
		return false
	}

	return true
}

func CreateUserWithQQ(qq int64) error {
	db := GetMySQLInstance().Database

	user := models.User{QQ: qq}

	err := db.Where(&user).FirstOrCreate(&user).Error
	return err
}
