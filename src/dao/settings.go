package dao

import (
	"avalon-core/src/dao/models"
	"time"
)

func GetSettings() (*models.Settings, error) {
	db := GetMySQLInstance().Database

	var Settings models.Settings
	err := db.First(&Settings).Error

	return &Settings, err
}

func SetBilibiliLastUpdate(t time.Time) error {
	db := GetMySQLInstance().Database

	var Settings models.Settings
	if err := db.First(&Settings).Error; err != nil {
		return err
	}

	Settings.BilibiliLastUpdate = t
	if err := db.Save(&Settings).Error; err != nil {
		return err
	}

	return nil

}
