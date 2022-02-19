package models

type Aria2Pending struct {
	ID      int    `gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY;"`
	Aria2Id string `gorm:"column:aria2id;NOT NULL;"`
	Name    string `gorm:"column:name;NOT NULL;"`
}

func (Aria2Pending) TableName() string {
	return "aria2_pending"
}
