package dao

import (
	"avalon-core/src/config"
	"avalon-core/src/dao/models"
	"avalon-core/src/utils"
	"errors"
	"gorm.io/gorm/logger"
	"log"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type singletonMySQL struct {
	Database *gorm.DB
}

var instance *singletonMySQL
var lock sync.Once

func GetMySQLInstance() *singletonMySQL {
	lock.Do(func() {
		// Init MySQL Instance
		db, err := gorm.Open(mysql.Open(utils.StringBuilder(config.Conf.GetString(`database.user`), `:`, config.Conf.GetString(`database.pass`), `@(`, config.Conf.GetString(`database.host`), `:`,
			config.Conf.GetString(`database.port`), `)/`, config.Conf.GetString(`database.name`), `?parseTime=true`)), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: false,
			Logger:                                   logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			log.Fatal(err)
		}

		instance = &singletonMySQL{
			Database: db,
		}
	})
	return instance
}

func InitDatabase() {
	db := GetMySQLInstance().Database
	err := db.Set(`gorm:table_options`, `ENGINE=InnoDB`).AutoMigrate(
		&models.Announcements{},
		&models.User{},
		&models.BilibiliEpisodeInfo{},
		&models.BilibiliArtist{},
		&models.BilibiliVideo{},
		&models.BilibiliSlice{},
		&models.BilibiliAudio{},
		&models.MikanSubtitleGroup{},
		&models.MikanAnimeSeries{},
		&models.MikanAnimeSet{},
		&models.MikanBlackListUrl{},
		&models.MikanResources{},
		&models.Aria2Pending{},
		&models.RuntimeControl{},
		&models.QQGroup{},
		&models.BilibiliArtistCollection{},
		&models.Settings{},
	)
	if err != nil {
		log.Println(err)
	}

	CheckSettings()
}

func CheckSettings() {
	db := GetMySQLInstance().Database
	var Settings models.Settings

	err := db.First(&Settings).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}

	Settings.BilibiliLastUpdate = time.Now()
	db.Create(&Settings)
}
