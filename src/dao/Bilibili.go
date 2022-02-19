package dao

import (
	"avalon-core/src/dao/models"
	"errors"
	"gorm.io/gorm/clause"
)

func FindOrCreateBilibiliEpisodeBySeasonTag(seasonTag string) (*models.BilibiliEpisodeInfo, error) {
	db := GetMySQLInstance().Database

	var episode models.BilibiliEpisodeInfo
	err := db.Preload(clause.Associations).Preload("Videos.Slices").Where(&models.BilibiliEpisodeInfo{SeasonTag: seasonTag}).FirstOrCreate(&episode).Error

	return &episode, err
}

func FindOrCreateArtistByName(name string) (*models.BilibiliArtist, error) {
	db := GetMySQLInstance().Database

	var artist models.BilibiliArtist
	err := db.Preload(clause.Associations).Preload("Videos.Slices").Where(&models.BilibiliArtist{Artist: name}).FirstOrCreate(&artist).Error

	return &artist, err
}

func FindOrCreateBilibiliVideoByBVID(BVID string) (*models.BilibiliVideo, error) {
	db := GetMySQLInstance().Database

	var video models.BilibiliVideo
	err := db.Preload(clause.Associations).Where(&models.BilibiliVideo{BVID: BVID}).FirstOrCreate(&video).Error

	return &video, err
}

func FindOrCreateCollectionByID(CollectionID string, ArtistID int) (*models.BilibiliArtistCollection, error) {
	db := GetMySQLInstance().Database

	var collection models.BilibiliArtistCollection
	err := db.Preload(clause.Associations).Preload("Videos.Slices").Where(&models.BilibiliArtistCollection{BilibiliID: CollectionID, ArtistID: ArtistID}).FirstOrCreate(&collection).Error

	return &collection, err
}

func SaveArtist(artist *models.BilibiliArtist) error {
	if artist == nil {
		return errors.New(`unexpected output`)
	}

	db := GetMySQLInstance().Database

	err := db.Preload(clause.Associations).Save(&artist).Error

	return err
}

func SaveCollection(collection *models.BilibiliArtistCollection) error {
	if collection == nil {
		return errors.New(`unexpected output`)
	}

	db := GetMySQLInstance().Database

	err := db.Preload(clause.Associations).Save(&collection).Error

	return err
}

func UpdateVideoCollection(video *models.BilibiliVideo, CollectionID int) error {
	if video == nil {
		return errors.New(`unexpected output`)
	}

	db := GetMySQLInstance().Database

	err := db.Model(video).Where(video).Update(`collection_id`, CollectionID).Error

	return err
}

func SaveVideo(video *models.BilibiliVideo) error {
	if video == nil {
		return errors.New(`unexpected output`)
	}

	db := GetMySQLInstance().Database

	err := db.Preload(clause.Associations).Save(&video).Error

	return err
}

func CreateBilibiliArtistVideo(artist *models.BilibiliArtist, videoID string) error {
	db := GetMySQLInstance().Database

	video, err := FindOrCreateBilibiliVideoByBVID(videoID)
	if err != nil {
		return err
	}

	artist.Videos = append(artist.Videos, *video)
	err = db.Preload(clause.Associations).Save(artist).Error
	return err
}

func SaveEpisode(episode *models.BilibiliEpisodeInfo) error {
	if episode == nil {
		return errors.New(`unexpected output`)
	}

	db := GetMySQLInstance().Database

	err := db.Preload(clause.Associations).Save(&episode).Error

	return err
}

func ListAllBilibiliSubscriptions() ([]models.BilibiliEpisodeInfo, error) {
	db := GetMySQLInstance().Database

	var indexList []int64
	var episodes []models.BilibiliEpisodeInfo

	if err := db.Table("subscription_bilibili").Select(`bilibili_episode_info_id`).Find(&indexList).Error; err != nil {
		return nil, err
	}

	if err := db.Where(`id IN ?`, indexList).Find(&episodes).Error; err != nil {
		return nil, err
	}

	return episodes, nil
}

func DeleteSubscription(episode *models.BilibiliEpisodeInfo) error {
	db := GetMySQLInstance().Database

	err := db.Table("subscription_bilibili").Where(`bilibili_episode_info_id = ?`, episode.ID).Delete(&struct{}{}).Error

	return err
}

func GetAllUsersByBilibiliSubscriptionID(id int) ([]models.User, error) {
	db := GetMySQLInstance().Database

	var users []models.User

	err := db.Model(&models.BilibiliEpisodeInfo{ID: id}).Association(`SubscribedUsers`).Find(&users)

	return users, err
}
