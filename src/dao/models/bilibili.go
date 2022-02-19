package models

type BilibiliArtist struct {
	ID          int                        `gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY;"`
	Artist      string                     `gorm:"column:artist;NOT NULL;"`
	Videos      []BilibiliVideo            `gorm:"foreignKey:ArtistID;"`
	Audios      []BilibiliAudio            `gorm:"foreignKey:ArtistID;"`
	Collections []BilibiliArtistCollection `gorm:"foreignKey:ArtistID;"`
	M3U8Link    string                     `gorm:"column:m3u8_link;NOT NULL;"`
	DirLink     string                     `gorm:"column:dir_link;NOT NULL;"`
}

func (BilibiliArtist) TableName() string {
	return "bilibili_artist"
}

type BilibiliAudio struct {
	ID       int    `gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY;"`
	AUID     string `gorm:"column:auid;NOT NULL;"`
	Title    string `gorm:"column:title;NOT NULL;"`
	ArtistID int    `gorm:"column:artist_id;NOT NULL;"`
	Link     string `gorm:"column:link;NOT NULL;"`
}

func (BilibiliAudio) TableName() string {
	return "bilibili_audio"
}

type BilibiliVideo struct {
	ID           int             `gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY;"`
	BVID         string          `gorm:"column:bvid;NOT NULL;"`
	Title        string          `gorm:"column:title;NOT NULL;"`
	ArtistID     int             `gorm:"column:artist_id;default:null"`
	EpisodeID    int             `gorm:"column:episode_id;default:null"`
	CollectionID int             `gorm:"column:collection_id;default:null"`
	Slices       []BilibiliSlice `gorm:"foreignKey:VideoID;"`
}

func (BilibiliVideo) TableName() string {
	return "bilibili_video"
}

type BilibiliArtistCollection struct {
	ID         int             `gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY;"`
	BilibiliID string          `gorm:"column:bilibili_id;NOT NULL;"`
	ArtistID   int             `gorm:"column:artist_id;NOT NULL;"`
	Title      string          `gorm:"column:title;NOT NULL;"`
	Videos     []BilibiliVideo `gorm:"foreignKey:CollectionID;"`
	M3U8Link   string          `gorm:"column:m3u8_link;NOT NULL;"`
}

func (BilibiliArtistCollection) TableName() string {
	return "bilibili_artist_collection"
}

type BilibiliEpisodeInfo struct {
	ID              int             `gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY;"`
	SeasonTag       string          `gorm:"column:season_tag;NOT NULL;"`
	Videos          []BilibiliVideo `gorm:"foreignKey:EpisodeID;"`
	M3U8Link        string          `gorm:"column:m3u8_link;NOT NULL;"`
	DirLink         string          `gorm:"column:dir_link;NOT NULL;"`
	SubscribedUsers []User          `gorm:"many2many:subscription_bilibili;"`
}

func (BilibiliEpisodeInfo) TableName() string {
	return "bilibili_episode_info"
}

type BilibiliSlice struct {
	Id       int64  `gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY;"`
	VideoID  string `gorm:"column:video_id;NOT NULL;"`
	CID      string `gorm:"column:cid;default:null"`
	SubTitle string `gorm:"column:subtitle;NOT NULL;"`
	Type     int8   `gorm:"column:type;NOT NULL;"` // 0 for ALL 1 for video only 2 for audio only
	Link     string `gorm:"column:link;NOT NULL;"`
}

func (BilibiliSlice) TableName() string {
	return "bilibili_slice"
}
