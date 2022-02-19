package models

type MikanAnimeSet struct {
	ID              int                 `gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY;"`
	AnimeID         int                 `gorm:"column:anime_id;NOT NULL;"`
	SubtitleGroupID int                 `gorm:"column:subtitle_group_id;NOT NULL;"`
	Resources       []MikanResources    `gorm:"foreignKey:BelongTo;"`
	BlackListUrls   []MikanBlackListUrl `gorm:"foreignKey:BelongTo;"`
	SubscribedUsers []User              `gorm:"many2many:subscription_mikan;"`
	M3U8Link        string              `gorm:"column:m3u8_link;NOT NULL;"`
	DirLink         string              `gorm:"column:dir_link;NOT NULL;"`
}

func (MikanAnimeSet) TableName() string {
	return "mikan_anime_set"
}

type MikanAnimeSeries struct {
	ID        int             `gorm:"column:id;NOT NULL;PRIMARY KEY;"`
	Name      string          `gorm:"column:name;NOT NULL;"`
	AnimeSets []MikanAnimeSet `gorm:"foreignKey:AnimeID;"`
}

func (MikanAnimeSeries) TableName() string {
	return "mikan_anime_series"
}

type MikanSubtitleGroup struct {
	ID        int             `gorm:"column:id;NOT NULL;PRIMARY KEY;"`
	Name      string          `gorm:"column:name;NOT NULL;"`
	AnimeSets []MikanAnimeSet `gorm:"foreignKey:SubtitleGroupID;"`
}

func (MikanSubtitleGroup) TableName() string {
	return "mikan_subtitle_group"
}

type MikanBlackListUrl struct {
	ID       int    `gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY;"`
	BelongTo int    `gorm:"column:belong_to;NOT NULL;"`
	URL      string `gorm:"column:url;NOT NULL;"`
}

func (MikanBlackListUrl) TableName() string {
	return "mikan_blacklist_url"
}

type MikanResources struct {
	ID       int    `gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY;"`
	Title    string `gorm:"column:title;NOT NULL;"`
	Link     string `gorm:"column:link;NOT NULL;"`
	IsDir    bool   `gorm:"column:link;NOT NULL;default:false"`
	BelongTo int    `gorm:"column:belong_to;NOT NULL;"`
}

func (MikanResources) TableName() string {
	return "mikan_resources"
}
