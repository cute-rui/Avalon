package dao

import (
	"avalon-core/src/dao/models"
	"avalon-core/src/utils"
	"errors"
	"gorm.io/gorm"
)

func FindOrCreatePending(gid, name string) (*models.Aria2Pending, bool, error) {
	db := GetMySQLInstance().Database

	var aria2 models.Aria2Pending
	err := db.Where(&models.Aria2Pending{Aria2Id: gid, Name: name}).First(&aria2).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		aria2.Aria2Id = gid
		aria2.Name = name
		err = db.Create(&aria2).Error
		if err != nil {
			return nil, false, err
		}

		return &aria2, false, nil
	} else if err != nil {
		return nil, false, err
	}

	return &aria2, true, nil
}

func FindOrCreatePendingByGID(gid string) (*models.Aria2Pending, bool, error) {
	db := GetMySQLInstance().Database

	var aria2 models.Aria2Pending
	err := db.Where(&models.Aria2Pending{Aria2Id: gid}).First(&aria2).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		aria2.Aria2Id = gid
		err = db.Create(&aria2).Error
		if err != nil {
			return nil, false, err
		}

		return &aria2, false, nil
	} else if err != nil {
		return nil, false, err
	}

	return &aria2, true, nil
}

func FindOrCreatePendingByName(name string) (*models.Aria2Pending, bool, error) {
	db := GetMySQLInstance().Database

	var aria2 models.Aria2Pending
	err := db.Where(`name LIKE ?`, utils.StringBuilder(`%`, name, `%`)).First(&aria2).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		aria2.Name = name
		err = db.Create(&aria2).Error
		if err != nil {
			return nil, false, err
		}

		return &aria2, false, nil
	} else if err != nil {
		return nil, false, err
	}

	return &aria2, true, nil
}

func FindPendingByName(name string) ([]models.Aria2Pending, error) {
	db := GetMySQLInstance().Database

	var aria2 []models.Aria2Pending
	err := db.Where(`name LIKE ?`, utils.StringBuilder(`%`, name, `%`)).Find(&aria2).Error

	return aria2, err
}

func CountPendingAll() (int, error) {
	db := GetMySQLInstance().Database

	var p int64
	err := db.Model(&models.Aria2Pending{}).Count(&p).Error
	return int(p), err
}

func CountPendingHTTP() (int, error) {
	db := GetMySQLInstance().Database

	var p int64
	err := db.Model(&models.Aria2Pending{}).Where(`name LIKE ?`, `%HTTP%`).Count(&p).Error
	return int(p), err
}

func DeletePendingByGID(gid string) error {
	db := GetMySQLInstance().Database

	var aria2 models.Aria2Pending
	err := db.Where(`aria2id = ?`, gid).Delete(&aria2).Error
	return err
}

func DeletePendingByName(name string) error {
	db := GetMySQLInstance().Database

	var aria2 models.Aria2Pending
	err := db.Where(`name = ?`, name).Delete(&aria2).Error
	return err
}

func Clean(gids []string) error {
	db := GetMySQLInstance().Database

	err := db.Where("aria2id IN ?", gids).Delete(&models.Aria2Pending{}).Error
	return err
}
