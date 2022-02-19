package dao

import (
	"avalon-core/src/dao/models"
	"avalon-core/src/utils"
)

func GetAria2FinishedByName(name string) bool {
	db := GetMySQLInstance().Database

	var runtime models.RuntimeControl
	db.Where(`data = ?`, name).First(&runtime)

	if runtime.Action == `ARIA2_FINISHED_NAME` {
		return true
	}

	return false

}

func NewFriendRequest(data string) error {
	db := GetMySQLInstance().Database

	runtime := models.RuntimeControl{
		Action: `NEW_FRIEND_REQUEST`,
		Data:   data,
	}
	return db.Create(&runtime).Error
}

func GetFriendRequest(token string) (*models.RuntimeControl, error) {
	db := GetMySQLInstance().Database

	runtime := models.RuntimeControl{}

	err := db.Model(&models.RuntimeControl{}).Where(`action = ?`, `NEW_FRIEND_REQUEST`).Where(`data LIKE ?`, utils.StringBuilder(`%`, token, `%`)).First(&runtime).Error
	return &runtime, err
}

func DeleteFriendRequest(token string) error {
	db := GetMySQLInstance().Database

	return db.Model(&models.RuntimeControl{}).Where(`action = ?`, `NEW_FRIEND_REQUEST`).Where(`data LIKE ?`, utils.StringBuilder(`%`, token, `%`)).Delete(&models.RuntimeControl{}).Error
}
