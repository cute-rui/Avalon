package dao

import (
	"avalon-core/src/dao/models"
	"errors"
	"gorm.io/gorm/clause"
)

func ListAllMikanSubscriptions() ([]models.MikanAnimeSet, error) {
	db := GetMySQLInstance().Database

	var indexList []int64
	var animeSets []models.MikanAnimeSet

	//todo cover
	if err := db.Table("subscription_mikan").Select(`mikan_anime_set_id`).Find(&indexList).Error; err != nil {
		return nil, err
	}

	if err := db.Preload(clause.Associations).Where(`id IN ?`, indexList).Find(&animeSets).Error; err != nil {
		return nil, err
	}

	return animeSets, nil
}

func ListAllAnimeSeries() ([]models.MikanAnimeSeries, error) {
	db := GetMySQLInstance().Database

	var animeSeries []models.MikanAnimeSeries

	err := db.Preload(clause.Associations).Preload(`AnimeSets.Resources`).Find(&animeSeries).Error

	return animeSeries, err
}

func FindOrCreateMikanSubtitleGroup(sid int) (*models.MikanSubtitleGroup, error) {
	db := GetMySQLInstance().Database

	var subgroup models.MikanSubtitleGroup
	err := db.Preload(clause.Associations).Where(&models.MikanSubtitleGroup{ID: sid}).FirstOrCreate(&subgroup).Error

	return &subgroup, err
}

func CreateMikanResource(r *models.MikanResources) (*models.MikanResources, error) {
	if r == nil {
		return nil, errors.New(`unexpected input`)
	}

	db := GetMySQLInstance().Database

	err := db.Create(&r).Error

	return r, err
}

func FindOrCreateMikanBangumi(bid int, title string) (*models.MikanAnimeSeries, error) {
	db := GetMySQLInstance().Database

	var bangumi models.MikanAnimeSeries
	err := db.Preload(clause.Associations).Where(&models.MikanAnimeSeries{ID: bid, Name: title}).FirstOrCreate(&bangumi).Error

	return &bangumi, err
}

func FindOrCreateMikanAnimeSet(bid, sid int, bangumiTitle string) (*models.MikanAnimeSet, error) {
	db := GetMySQLInstance().Database

	var animeSet models.MikanAnimeSet

	if _, err := FindOrCreateMikanSubtitleGroup(sid); err != nil {
		return nil, err
	}

	if _, err := FindOrCreateMikanBangumi(bid, bangumiTitle); err != nil {
		return nil, err
	}

	err := db.Preload(clause.Associations).Where(&models.MikanAnimeSet{
		AnimeID:         bid,
		SubtitleGroupID: sid,
	}).FirstOrCreate(&animeSet).Error

	return &animeSet, err
}

func GetAllUsersByMikanSubscriptionID(id int) ([]models.User, error) {
	db := GetMySQLInstance().Database

	var users []models.User

	err := db.Model(&models.MikanAnimeSet{ID: id}).Association(`SubscribedUsers`).Find(&users)

	return users, err
}

func SaveMikanAnimeSet(anime *models.MikanAnimeSet) error {
	if anime == nil {
		return errors.New(`unexpected output`)
	}

	db := GetMySQLInstance().Database

	err := db.Save(&anime).Error

	return err
}

func FindAnimeSetByIDS(bid, sid int) (*models.MikanAnimeSet, error) {
	db := GetMySQLInstance().Database

	var anime models.MikanAnimeSet

	err := db.Preload(clause.Associations).Where(&models.MikanAnimeSet{AnimeID: bid, SubtitleGroupID: sid}).First(&anime).Error
	return &anime, err
}
