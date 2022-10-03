package Moefunc

import (
	"avalon-core/src/config"
	"avalon-core/src/dao"
	"avalon-core/src/dao/models"
	"avalon-core/src/function/aria2"
	"avalon-core/src/function/bilibili"
	"avalon-core/src/function/ffmpeg"
	"avalon-core/src/function/fileSystemService"
	"avalon-core/src/log"
	"avalon-core/src/utils"
	"errors"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
)

var PendingTask map[string][]chan []string
var TaskLock sync.Mutex

func init() {
	PendingTask = make(map[string][]chan []string)
}

func BilibiliUniversalDownload(user *models.User, data string) string {
	VideoList, err := DoBilibiliDataQuery(data, true)
	if err != nil {
		return ``
	}

	switch VideoList.Type {
	case bilibili.DataType_Season:
		return BilibiliSubscribe(VideoList, user)
	case bilibili.DataType_Video:
		Video := BilibiliSingleVideoDownload(VideoList)
		if Video == nil {
			return ``
		} else {
			return GetSingleVideoResponse(Video)
		}
	case bilibili.DataType_Collection:
		return BilibiliCollectionDownload(VideoList, data)
	}

	return ``
}

func BilibiliSingleVideoDownload(VideoList *bilibili.ParsedQuery) *models.BilibiliVideo {
	if VideoList == nil {
		return nil
	}

	updates := CheckAndWaitTaskFinished(VideoList.ID)

	defer SendCloseSig(VideoList.ID)

	artist, err := dao.FindOrCreateArtistByName(VideoList.Author)
	if err != nil {
		log.Logger.Error(err)
	}

	for i := range artist.Videos {
		if artist.Videos[i].BVID == VideoList.ID {
			needUpdate, u := IsSliceNeedUpdateByCheckingCID(artist.Videos[i].Slices, VideoList)

			if len(u) != 0 {
				needUpdate = true
				updates = append(updates, u...)
			}

			if !needUpdate {
				SendSuccessBroadCast(VideoList.ID, updates)
				return &artist.Videos[i]
			}
		}
	}

	if len(updates) == 0 {
		updates = nil
	}
	drive, err := GetBilibiliRcloneDrive()
	if drive == `` || err != nil {
		return nil
	}

	BilibiliVideos, err := DoBilibiliSingleVideoDownload(drive, updates, VideoList)
	if err != nil {
		log.Logger.Error(err)
		return nil
	} else if BilibiliVideos == nil {
		return nil
	}

	index, err := UpdateArtistVideoInfo(artist, BilibiliVideos)
	if err != nil {
		log.Logger.Error(err)
	}

	M3UList, err := GetArtistM3UList(artist)
	if err != nil {
		log.Logger.Error(err)
		return nil
	}

	link, err := UploadBilibiliM3UFile(drive, VideoList.Author, M3UList)
	if err != nil {
		log.Logger.Error(err)
		return nil
	}

	artist.M3U8Link = link

	err = dao.SaveArtist(artist)
	if err != nil {
		log.Logger.Error(err)
		return nil
	}

	SendSuccessBroadCast(VideoList.ID, updates)

	return &artist.Videos[index]
}

func BilibiliUpdateCollectionVideo(VideoList *bilibili.ParsedQuery) *models.BilibiliVideo {
	if VideoList == nil {
		return nil
	}

	updates := CheckAndWaitTaskFinished(VideoList.ID)

	defer SendCloseSig(VideoList.ID)

	artist, err := dao.FindOrCreateArtistByName(VideoList.Author)
	if err != nil {
		log.Logger.Error(err)
	}

	for i := range artist.Videos {
		if artist.Videos[i].BVID == VideoList.ID {
			needUpdate, u := IsSliceNeedUpdateByCheckingCID(artist.Videos[i].Slices, VideoList)

			if len(u) != 0 {
				needUpdate = true
				updates = append(updates, u...)
			}

			if !needUpdate {
				SendSuccessBroadCast(VideoList.ID, updates)
				return &artist.Videos[i]
			}
		}
	}

	if len(updates) == 0 {
		updates = nil
	}
	drive, err := GetBilibiliRcloneDrive()
	if drive == `` || err != nil {
		return nil
	}

	BilibiliVideos, err := DoBilibiliSingleVideoDownload(drive, updates, VideoList)
	if err != nil {
		log.Logger.Error(err)
		return nil
	} else if BilibiliVideos == nil {
		return nil
	}

	index, err := UpdateArtistVideoInfo(artist, BilibiliVideos)
	if err != nil {
		log.Logger.Error(err)
	}

	err = dao.SaveArtist(artist)
	if err != nil {
		log.Logger.Error(err)
		return nil
	}

	SendSuccessBroadCast(VideoList.ID, updates)

	return &artist.Videos[index]
}

func BilibiliCollectionDownload(VideoList *bilibili.ParsedQuery, data string) string {
	if VideoList == nil {
		return ``
	}

	updates := CheckAndWaitTaskFinished(VideoList.ID)

	defer SendCloseSig(VideoList.ID)

	artist, err := dao.FindOrCreateArtistByName(VideoList.Author)
	if err != nil {
		log.Logger.Error(err)
		return ``
	}

	collection, err := dao.FindOrCreateCollectionByID(VideoList.ID, artist.ID)
	if err != nil {
		log.Logger.Error(err)
		return ``
	}

	if !IsCollectionNeedUpdateByCheckingCID(collection, VideoList) {
		return GetCollectionVideoResponse(collection)
	}
	videoInfoMap := map[string]*bilibili.ParsedQuery{}
	videoListMap := map[string]*models.BilibiliVideo{}
	for i := range VideoList.Parts {
		if v, ok := videoInfoMap[VideoList.Parts[i].BVIDInCollection]; ok && v != nil {
			v.Parts = append(v.Parts, VideoList.Parts[i])
		} else {
			v := bilibili.ParsedQuery{
				Type:   bilibili.DataType_Video,
				ID:     VideoList.Parts[i].BVIDInCollection,
				Author: VideoList.Author,
			}
			v.Parts = append(v.Parts, VideoList.Parts[i])
			videoInfoMap[VideoList.Parts[i].BVIDInCollection] = &v
		}
	}

	var WG sync.WaitGroup
	for _, v := range videoInfoMap {
		WG.Add(1)
		go func(VideoList *bilibili.ParsedQuery) {
			Video := BilibiliUpdateCollectionVideo(VideoList)
			videoListMap[data] = Video
			WG.Done()
		}(v)
	}

	WG.Wait()

	for k, v := range videoListMap {
		WG.Add(1)
		go func(k string, v *models.BilibiliVideo) {
			if v == nil {
				log.Logger.Error(`error on updating collection #`, k)
				return
			}

			err := dao.UpdateVideoCollection(v, collection.ID)
			if err != nil {
				log.Logger.Error(err)
			}

			WG.Done()
		}(k, v)
	}

	WG.Wait()

	drive, err := GetBilibiliRcloneDrive()
	if drive == `` || err != nil {
		return ``
	}

	artist, err = dao.FindOrCreateArtistByName(VideoList.Author)
	if err != nil {
		log.Logger.Error(err)
	}

	collection, err = dao.FindOrCreateCollectionByID(VideoList.ID, artist.ID)
	if err != nil {
		log.Logger.Error(err)
	}

	ArtistM3UList, err := GetArtistM3UList(artist)
	if err != nil {
		log.Logger.Error(err)
		return ``
	}

	CollectionM3UList, err := GetCollectionM3UList(collection)
	if err != nil {
		log.Logger.Error(err)
		return ``
	}

	link, err := UploadBilibiliM3UFile(drive, VideoList.Author, ArtistM3UList)
	if err != nil {
		log.Logger.Error(err)
		return ``
	}

	artist.M3U8Link = link

	err = dao.SaveArtist(artist)
	if err != nil {
		log.Logger.Error(err)
		return ``
	}

	link, err = UploadBilibiliM3UFile(drive, VideoList.Author, CollectionM3UList)
	if err != nil {
		log.Logger.Error(err)
		return ``
	}

	collection.M3U8Link = link
	collection.Title = VideoList.CollectionTitle
	err = dao.SaveCollection(collection)
	if err != nil {
		log.Logger.Error(err)
		return ``
	}

	SendSuccessBroadCast(VideoList.ID, updates)

	return GetCollectionVideoResponse(collection)

}

func BilibiliSubscribe(VideoList *bilibili.ParsedQuery, user *models.User) string {
	if VideoList == nil {
		return ``
	}

	var exist bool
	for i := range user.Bilibili {
		if user.Bilibili[i].SeasonTag != VideoList.ID {
			continue
		}

		exist = true
	}

	if !VideoList.IsEnd && !exist {
		err := dao.CreateBilibiliSubscription(user, VideoList.ID)
		if err != nil {
			log.Logger.Error(err)
			return ``
		}
	}

	updates := CheckAndWaitTaskFinished(VideoList.ID)

	defer SendCloseSig(VideoList.ID)

	ep, err := dao.FindOrCreateBilibiliEpisodeBySeasonTag(VideoList.ID)
	if err != nil {
		log.Logger.Error(err)
	}

	needUpdate, u := IsEpisodeNeedUpdateByCheckingBVID(ep, VideoList)

	if len(u) != 0 {
		needUpdate = true
		updates = append(updates, u...)
	}

	if !needUpdate {
		SendSuccessBroadCast(VideoList.ID, updates)

		return GetEpisodeResponse(ep)
	}

	drive, err := GetBilibiliRcloneDrive()
	if drive == `` || err != nil {
		return ``
	}

	BilibiliVideos, err := DoBilibiliEpisodeDownload(drive, updates, VideoList)
	if err != nil {
		log.Logger.Error(err)
		return ``
	} else if BilibiliVideos == nil {
		return ``
	}

	UpdateEpisodeInfo(ep, BilibiliVideos, VideoList.CollectionTitle)

	M3UList, err := GetEpisodeM3UList(ep)
	if err != nil {
		log.Logger.Error(err)
		return ``
	}

	link, err := UploadBilibiliM3UFile(drive, VideoList.CollectionTitle, M3UList)
	if err != nil {
		log.Logger.Error(err)
		return ``
	}

	ep.M3U8Link = link

	err = dao.SaveEpisode(ep)
	if err != nil {
		log.Logger.Error(err)
		return ``
	}

	SendSuccessBroadCast(VideoList.ID, updates)

	return GetEpisodeResponse(ep)
}

func BilibiliSubscribeUpdate(Season string) {
	VideoList, err := DoBilibiliDataQuery(utils.StringBuilder(`https://www.bilibili.com/bangumi/play/ss`, Season), false)
	if err != nil {
		return
	}

	updates := CheckAndWaitTaskFinished(VideoList.ID)

	defer SendCloseSig(VideoList.ID)

	ep, err := dao.FindOrCreateBilibiliEpisodeBySeasonTag(VideoList.ID)
	if err != nil {
		log.Logger.Error(err)
	}

	needUpdate, u := IsEpisodeNeedUpdateByCheckingBVID(ep, VideoList)

	if len(u) != 0 {
		needUpdate = true
		updates = append(updates, u...)
	}

	if !needUpdate {
		SendSuccessBroadCast(VideoList.ID, updates)
		return
	}

	drive, err := GetBilibiliRcloneDrive()
	if drive == `` || err != nil {
		return
	}

	BilibiliVideos, err := DoBilibiliEpisodeDownload(drive, updates, VideoList)
	if err != nil {
		log.Logger.Error(err)
		return
	} else if BilibiliVideos == nil {
		return
	}

	UpdateEpisodeInfo(ep, BilibiliVideos, VideoList.CollectionTitle)

	M3UList, err := GetEpisodeM3UList(ep)
	if err != nil {
		log.Logger.Error(err)
		return
	}

	link, err := UploadBilibiliM3UFile(drive, VideoList.CollectionTitle, M3UList)
	if err != nil {
		log.Logger.Error(err)
		return
	}

	ep.M3U8Link = link

	err = dao.SaveEpisode(ep)
	if err != nil {
		log.Logger.Error(err)
		return
	}

	err = BilibiliSubscriptionSendMiraiBroadCast(utils.StringBuilder(`您订阅的`, VideoList.CollectionTitle, "已更新:\n", GetUpdatedEpisodeResponse(ep, updates...)), ep.ID)
	if err != nil {
		log.Logger.Error(err)
		return
	}

	SendSuccessBroadCast(VideoList.ID, updates)
	return
}

func IsEpisodeNeedUpdateByCheckingBVID(ep *models.BilibiliEpisodeInfo, VideoList *bilibili.ParsedQuery) (bool, []string) {
	m := map[string]int{}
	s := []string{}
	if ep == nil || VideoList == nil {
		return false, s
	}

	for i := range ep.Videos {
		m[ep.Videos[i].BVID]++
	}

	for i := range VideoList.Parts {
		k, _ := m[VideoList.Parts[i].BVIDInCollection]
		if k < 1 {
			m[VideoList.Parts[i].BVIDInCollection]++
			s = append(s, VideoList.Parts[i].BVIDInCollection)
		}
	}

	if len(s) != 0 {
		return true, s
	}

	return false, s
}

func IsCollectionNeedUpdateByCheckingCID(collection *models.BilibiliArtistCollection, VideoList *bilibili.ParsedQuery) bool {
	if collection == nil || VideoList == nil {
		return true
	}

	//VideoMap := map[string]int{}
	SliceMap := map[string]int{}

	for i := range collection.Videos {
		//VideoMap[collection.Videos[i].BVID] = i
		for j := range collection.Videos[i].Slices {
			SliceMap[collection.Videos[i].Slices[j].CID] = j
		}
	}

	for i := range VideoList.Parts {
		//if _, ok := VideoMap[VideoList.Parts[i].BVIDInCollection]; !ok {
		//    return true
		//} else
		if _, ok := SliceMap[VideoList.Parts[i].CIDOrEPID]; !ok {
			return true
		}
	}

	return false
}

func IsSliceNeedUpdateByCheckingCID(slices []models.BilibiliSlice, VideoList *bilibili.ParsedQuery) (bool, []string) {
	m := map[string]int{}
	s := []string{}

	if VideoList == nil {
		return false, s
	}

	for i := range slices {
		m[slices[i].CID]++
	}

	for i := range VideoList.Parts {
		k, _ := m[VideoList.Parts[i].CIDOrEPID]
		if k < 1 {
			m[VideoList.Parts[i].CIDOrEPID]++
			s = append(s, VideoList.Parts[i].CIDOrEPID)
		}
	}

	if len(s) != 0 {
		return true, s
	}

	return false, s
}

func CheckAndWaitTaskFinished(ID string) []string {
	if _, ok := PendingTask[ID]; ok {
		ch := make(chan []string)
		PendingTask[ID] = append(PendingTask[ID], ch)
		select {
		case s := <-ch:
			if s == nil || len(s) == 0 {
				return []string{}
			} else {
				return s
			}
		}
	} else {
		PendingTask[ID] = nil
		return []string{}
	}
}

func DoBilibiliEpisodeDownload(drive string, targets []string, infos *bilibili.ParsedQuery) ([]*BilibiliVideo, error) {
	var (
		WG             sync.WaitGroup
		BilibiliVideos []*BilibiliVideo
	)

	targetMap := make(map[string]int)
	for i := range targets {
		targetMap[targets[i]]++
	}

	var Region bilibili.Region
	for i := range infos.Parts {
		if _, ok := targetMap[infos.Parts[i].BVIDInCollection]; !ok {
			continue
		}

		Videos, err := GetBilibiliDownloadInfo(``, infos.Parts[i].CIDOrEPID, infos.CollectionTitle, bilibili.DataType_Season, infos.Parts[i].Region)
		Region = infos.Parts[i].Region
		if err != nil {
			return nil, err
		}

		for j := range Videos {
			Videos[j].EpisodeID = infos.Parts[i].CIDOrEPID
			Videos[j].Region = infos.Parts[i].Region
			BilibiliVideos = append(BilibiliVideos, Videos[j])
		}
	}

	log.Logger.Debug(`updating: `, infos.CollectionTitle)

	for index := range BilibiliVideos {
		if BilibiliVideos[index] == nil {
			continue
		}

		WG.Add(1)
		go func(i int) {
			for {
				count, err := dao.CountPendingHTTP()

				if err != nil {
					log.Logger.Error(err)
				}

				if count >= config.Conf.GetInt(`worker.aria2.settings.maxDownloadTasks`) {
					time.Sleep(1 * time.Minute)
				} else {
					break
				}
			}

			err := GetVideoAndAudioSource(BilibiliVideos[i].EpisodeID, bilibili.DataType_Season, Region, BilibiliVideos[i])
			if err != nil {
				log.Logger.Error(err)
				WG.Done()
				return
			}

			/*Video := BilibiliVideo{
			    ID:       VideoList.Parts[i].VideoID,
			    CID:      VideoList.Parts[i].CID,
			    DirName:  VideoList.CollectionTitle,
			    SubTitle: VideoList.Parts[i].SubTitle,
			    FileName: utils.StringBuilder(VideoList.CollectionTitle, ` - `, VideoList.Parts[i].SubTitle, ` - `, VideoList.Parts[i].VideoID, ` - `, VideoList.Parts[i].Quality, `.mkv`),
			    Audio:    a,
			    Video:    v,
			}*/

			err = BilibiliVideos[i].DownloadAndUpload(drive)
			if err != nil {
				log.Logger.Error(err)
			}

			if BilibiliVideos[i].DownloadStatus != 2 {
				log.Logger.Error(`error in donwloading video:`, BilibiliVideos[i].SubTitle, `-`, BilibiliVideos[i].ID, `-`, BilibiliVideos[i].CID)
			}

			WG.Done()
		}(index)
		time.Sleep(10 * time.Second)
	}
	WG.Wait()

	return CheckIfBiliVideoValid(BilibiliVideos), nil
}

func DoBilibiliSingleVideoDownload(drive string, targets []string, infos *bilibili.ParsedQuery) ([]*BilibiliVideo, error) {
	var (
		WG             sync.WaitGroup
		BilibiliVideos []*BilibiliVideo
	)

	targetMap := make(map[string]int)
	for i := range targets {
		targetMap[targets[i]]++
	}

	if strings.HasPrefix(infos.ID, `BV`) {
		for i := range infos.Parts {
			if _, ok := targetMap[infos.Parts[i].CIDOrEPID]; !ok {
				continue
			}

			Videos, err := GetBilibiliDownloadInfo(infos.Parts[i].CIDOrEPID, infos.ID, infos.Author, bilibili.DataType_Video, bilibili.Region_CN)

			if err != nil {
				return nil, err
			}

			for j := range Videos {
				BilibiliVideos = append(BilibiliVideos, Videos[j])
			}
		}
	} else {
		for i := range infos.Parts {
			if _, ok := targetMap[infos.Parts[i].CIDOrEPID]; !ok {
				continue
			}

			Videos, err := GetBilibiliDownloadInfo(infos.Parts[i].CIDOrEPID, infos.Parts[i].BVIDInCollection, infos.Parts[i].Author, bilibili.DataType_Video, bilibili.Region_CN)

			if err != nil {
				return nil, err
			}

			for j := range Videos {
				BilibiliVideos = append(BilibiliVideos, Videos[j])
			}
		}
	}

	for index := range BilibiliVideos {
		WG.Add(1)
		go func(i int) {
			for {
				count, err := dao.CountPendingHTTP()

				if err != nil {
					log.Logger.Error(err)
				}

				if count >= config.Conf.GetInt(`worker.aria2.settings.maxDownloadTasks`) {
					time.Sleep(1 * time.Minute)
				} else {
					break
				}
			}

			err := GetVideoAndAudioSource(BilibiliVideos[i].ID, bilibili.DataType_Video, bilibili.Region_CN, BilibiliVideos[i])
			if err != nil {
				log.Logger.Error(err)
				WG.Done()
				return
			}

			/*Video := BilibiliVideo{
			    ID:       VideoList.Parts[i].VideoID,
			    CID:      VideoList.Parts[i].CID,
			    DirName:  VideoList.Author,
			    Title:    VideoList.Parts[i].Title,
			    SubTitle: VideoList.Parts[i].SubTitle,
			    FileName: utils.StringBuilder(VideoList.Parts[i].Title, ` - `, VideoList.Parts[i].SubTitle, ` - `, VideoList.Parts[i].VideoID, ` - `, VideoList.Parts[i].Quality, `.mkv`),
			    Audio:    a,
			    Video:    v,
			}*/

			err = BilibiliVideos[i].DownloadAndUpload(drive)
			if err != nil {
				log.Logger.Error(err)
			}

			if BilibiliVideos[i].DownloadStatus != 2 {
				log.Logger.Error(`error in donwloading video:`, BilibiliVideos[i].SubTitle, `-`, BilibiliVideos[i].ID, `-`, BilibiliVideos[i].CID)
			}

			WG.Done()
		}(index)
		time.Sleep(10 * time.Second)
	}
	WG.Wait()

	return CheckIfBiliVideoValid(BilibiliVideos), nil
}

func CheckIfBiliVideoValid(InputVideos []*BilibiliVideo) []*BilibiliVideo {
	var OutputVideos []*BilibiliVideo
	for i := range InputVideos {
		if InputVideos[i].RcloneFileLink == `` || InputVideos[i].DownloadStatus != 2 {
			continue
		}
		OutputVideos = append(OutputVideos, InputVideos[i])
	}

	return OutputVideos
}

func GetBilibiliRcloneDrive() (string, error) {
	var (
		drive string
		err   error
		retry = 1
	)
	if config.Conf.GetInt(`worker.fs.settings.Retry`) > 0 {
		retry = config.Conf.GetInt(`worker.fs.settings.Retry`)
	}

	for i := 0; i <= retry; i++ {
		drive, err = fileSystemService.RcloneGetDrive(`bilibili`)
		if err == nil && drive != `` {
			return drive, nil
		}
		log.Logger.Error(err)
	}

	return ``, err
}

func GetVideoAndAudioSource(BVorEPID string, Type bilibili.DataType, Region bilibili.Region, video *BilibiliVideo) error {
	VideoList, err := GetBilibiliDownloadInfo(video.CID, BVorEPID, video.DirName, Type, Region)
	if err != nil {
		return err
	}

	for i := range VideoList {
		if video.CID == VideoList[i].CID && video.SubTitle == VideoList[i].SubTitle {
			return nil
		}
	}

	return errors.New(utils.StringBuilder(`no video info matched on:`, video.SubTitle, `-`, BVorEPID, `-`, video.CID))
}

/*func DoVideoTypeMatching() {
    if utils.StringBuilder(v.Parts[j].CID, v.Parts[j].SubTitle) == targets[i] {
        t = append(t, v.Parts[j])
    }
}

func DoSeasonTypeMatching() {
    if utils.StringBuilder(v.Parts[j].VideoID, v.Parts[j].SubTitle) == targets[i] {
        t = append(t, v.Parts[j])
    }
}*/

func DoBilibiliDataQuery(url string, checkCollection bool) (*bilibili.ParsedQuery, error) {
	var (
		v     *bilibili.ParsedQuery
		err   error
		retry = 1
	)

	if config.Conf.GetInt(`worker.bilibili.settings.Retry`) > 0 {
		retry = config.Conf.GetInt(`worker.bilibili.settings.Retry`)
	}

	for times := 0; times <= retry; times++ {
		v, err = bilibili.DoDownloadQuery(url, checkCollection)
		if err == nil && v != nil {
			return v, nil
		}
		log.Logger.Error(err)
	}

	return nil, err
}

func GetBilibiliDownloadInfo(CID, BVorEPID, DirName string, Type bilibili.DataType, Region bilibili.Region) ([]*BilibiliVideo, error) {
	var (
		VideoInfoList []*bilibili.ParsedPartInfo
		result        []*BilibiliVideo
		err           error
		retry         = 1
	)

	if config.Conf.GetInt(`worker.bilibili.settings.Retry`) > 0 {
		retry = config.Conf.GetInt(`worker.bilibili.settings.Retry`)
	}

	for times := 0; times <= retry; times++ {
		VideoInfoList, err = bilibili.GetDownloadLink(CID, BVorEPID, Type, Region)
		if err == nil && VideoInfoList != nil {
			switch Type {
			case bilibili.DataType_Video:
				for i := range VideoInfoList {
					result = append(result, &BilibiliVideo{
						ID:        VideoInfoList[i].VideoID,
						CID:       VideoInfoList[i].CID,
						DirName:   DirName, //author
						Title:     VideoInfoList[i].Title,
						SubTitle:  VideoInfoList[i].SubTitle,
						FileName:  utils.StringBuilder(VideoInfoList[i].Title, ` - `, VideoInfoList[i].SubTitle, ` - `, VideoInfoList[i].VideoID, ` - `, VideoInfoList[i].Quality, `.mkv`),
						Audio:     VideoInfoList[i].AudioSource,
						Video:     VideoInfoList[i].VideoSource,
						Subtitles: VideoInfoList[i].SubtitleInfos,
					})
				}

			case bilibili.DataType_Season:
				for i := range VideoInfoList {
					result = append(result, &BilibiliVideo{
						ID:        VideoInfoList[i].VideoID,
						CID:       VideoInfoList[i].CID,
						DirName:   DirName, //collection
						Title:     VideoInfoList[i].Title,
						SubTitle:  VideoInfoList[i].SubTitle,
						FileName:  utils.StringBuilder(DirName, ` - `, VideoInfoList[i].SubTitle, ` - `, VideoInfoList[i].VideoID, ` - `, VideoInfoList[i].Quality, `.mkv`),
						Audio:     VideoInfoList[i].AudioSource,
						Video:     VideoInfoList[i].VideoSource,
						Subtitles: VideoInfoList[i].SubtitleInfos,
					})
				}
			}

			return result, nil
		}
		log.Logger.Error(err)
	}

	return nil, err
}

func UpdateEpisodeInfo(ep *models.BilibiliEpisodeInfo, BilibiliVideos []*BilibiliVideo, title string) {
	if ep == nil {
		return
	}

	m := map[string]int{}
	for i := range ep.Videos {
		m[ep.Videos[i].BVID] = i
	}

	for i := range BilibiliVideos {
		if BilibiliVideos[i] != nil {
			if v, ok := m[BilibiliVideos[i].ID]; ok {
				ep.Videos[v].Slices = append(ep.Videos[v].Slices, models.BilibiliSlice{
					SubTitle: BilibiliVideos[i].SubTitle,
					Type:     0,
					Link:     BilibiliVideos[i].RcloneFileLink,
				})
			} else {
				ep.Videos = append(ep.Videos, models.BilibiliVideo{
					BVID:      BilibiliVideos[i].ID,
					Title:     title,
					EpisodeID: ep.ID,
					Slices: []models.BilibiliSlice{{
						SubTitle: BilibiliVideos[i].SubTitle,
						Type:     0,
						Link:     BilibiliVideos[i].RcloneFileLink,
					}},
				})
			}
		}
	}
}

func UpdateArtistVideoInfo(artist *models.BilibiliArtist, BilibiliVideos []*BilibiliVideo) (int, error) {
	if artist == nil {
		return 0, errors.New(`unexpected artist`)
	}

	index := 0
	VideoMap := map[string]int{}
	SliceMap := map[string]int{}

	for i := range artist.Videos {
		VideoMap[artist.Videos[i].BVID] = i
		for j := range artist.Videos[i].Slices {
			SliceMap[utils.StringBuilder(artist.Videos[i].Slices[j].SubTitle, artist.Videos[i].Slices[j].CID)] = j
		}
	}

	for i := range BilibiliVideos {
		if _, ok := VideoMap[BilibiliVideos[i].ID]; !ok {
			artist.Videos = append(artist.Videos, models.BilibiliVideo{
				ID:        0,
				BVID:      BilibiliVideos[i].ID,
				Title:     BilibiliVideos[i].Title,
				ArtistID:  artist.ID,
				EpisodeID: 0,
				Slices: []models.BilibiliSlice{{
					CID:      BilibiliVideos[i].CID,
					SubTitle: BilibiliVideos[i].SubTitle,
					Type:     0,
					Link:     BilibiliVideos[i].RcloneFileLink,
				}},
			})
			VideoMap[BilibiliVideos[i].ID] = len(artist.Videos) - 1
			index = len(artist.Videos) - 1
		} else if _, ok := SliceMap[utils.StringBuilder(BilibiliVideos[i].SubTitle, BilibiliVideos[i].CID)]; !ok {
			index = VideoMap[BilibiliVideos[i].ID]
			artist.Videos[index].Slices = append(artist.Videos[index].Slices, models.BilibiliSlice{
				CID:      BilibiliVideos[i].CID,
				SubTitle: BilibiliVideos[i].SubTitle,
				Type:     0,
				Link:     BilibiliVideos[i].RcloneFileLink,
			})
		}
	}

	return index, nil

}

/*func UpdateArtistCollectionInfo(artist *models.BilibiliArtist, BilibiliVideos []*BilibiliVideo, CollectionTitle string) int {
    if artist == nil {
        return -1
    }

    for j := range artist.Collections {
        if artist.Collections[j].Title == CollectionTitle {
            for i := range BilibiliVideos {
                if BilibiliVideos[i] != nil {
                    for k := range artist.Collections[j].Videos {
                        if artist.Collections[j].Videos[k].BVID == BilibiliVideos[i].ID

                    }
                    artist.Collections[j].Videos = append(artist.Videos[j].Slices, models.BilibiliSlice{
                        VideoID:  BilibiliVideos[i].ID,
                        CID:      BilibiliVideos[i].CID,
                        SubTitle: BilibiliVideos[i].SubTitle,
                        Type:     0,
                        Link:     BilibiliVideos[i].RcloneFileLink,
                    })
                }
            }
            return j
        }
    }

    return -1

}*/

func GetEpisodeResponse(ep *models.BilibiliEpisodeInfo) string {
	if ep == nil {
		return ``
	}

	S := []string{}
	for i := range ep.Videos {
		for j := range ep.Videos[i].Slices {
			S = append(S, utils.StringBuilder(ep.Videos[i].Title, ` - `, ep.Videos[i].Slices[j].SubTitle, ":\n", ep.Videos[i].Slices[j].Link, "\n"))
			sort.Strings(S)
		}
	}

	return utils.StringBuilder(utils.StringBuilder(S...), "m3u剧集列表：\n", ep.M3U8Link)
}

func GetCollectionVideoResponse(collection *models.BilibiliArtistCollection) string {
	if collection == nil {
		return ``
	}

	if len(collection.Videos) == 0 {
		return ``
	}

	S := []string{}
	for i := range collection.Videos {
		for j := range collection.Videos[i].Slices {
			S = append(S, utils.StringBuilder(collection.Videos[i].Title, ` - `, collection.Videos[i].Slices[j].SubTitle, ":\n", collection.Videos[i].Slices[j].Link, "\n"))
		}

	}

	sort.Strings(S)
	return utils.StringBuilder(utils.StringBuilder(S...), "m3u剧集列表：\n", collection.M3U8Link)
}

func GetSingleVideoResponse(video *models.BilibiliVideo) string {
	if video == nil {
		return ``
	}

	S := []string{}
	for i := range video.Slices {
		S = append(S, utils.StringBuilder(video.Title, ` - `, video.Slices[i].SubTitle, ":\n", video.Slices[i].Link, "\n"))
	}

	sort.Strings(S)
	return utils.StringBuilder(S...)
}

func GetUpdatedEpisodeResponse(ep *models.BilibiliEpisodeInfo, targetsID ...string) string {
	if ep == nil {
		return ``
	}

	S := []string{}
	for i := range ep.Videos {
		for j := range ep.Videos[i].Slices {
			if targetsID == nil || len(targetsID) == 0 {
				S = append(S, utils.StringBuilder(ep.Videos[i].Title, ` - `, ep.Videos[i].Slices[j].SubTitle, ":\n", ep.Videos[i].Slices[j].Link, "\n"))
			} else {
				for k := range targetsID {
					if targetsID[k] == ep.Videos[i].BVID {
						S = append(S, utils.StringBuilder(ep.Videos[i].Title, ` - `, ep.Videos[i].Slices[j].SubTitle, ":\n", ep.Videos[i].Slices[j].Link, "\n"))
						break
					}
				}
			}
		}
	}

	sort.Strings(S)
	return utils.StringBuilder(utils.StringBuilder(S...), "m3u剧集列表：\n", ep.M3U8Link)
}

func SendSuccessBroadCast(ID string, data []string) {
	TaskLock.Lock()
	if _, ok := PendingTask[ID]; ok {
		for i := range PendingTask[ID] {
			PendingTask[ID][i] <- data
		}
		delete(PendingTask, ID)
	}
	TaskLock.Unlock()
}

func SendCloseSig(ID string) {
	if _, ok := PendingTask[ID]; ok {
		if len(PendingTask[ID]) == 0 {
			delete(PendingTask, ID)
			return
		}

		for i := range PendingTask[ID] {
			PendingTask[ID][i] <- nil
			break
		}
	}
}

type BilibiliVideo struct {
	ID             string
	CID            string
	DirName        string //season name or artist
	Title          string
	SubTitle       string
	FileName       string
	Audio          string
	Video          string
	Subtitles      []*bilibili.SubtitleInfo
	Region         bilibili.Region
	EpisodeID      string
	RcloneFileLink string
	DownloadStatus int //0 -> not have a task 1 -> have a task but unfinished 2 -> finished
}

func (b *BilibiliVideo) DownloadAndUpload(remoteDrive string) error {
	if strings.Contains(b.DirName, "/") {
		b.DirName = strings.Replace(b.DirName, "/", "-", -1)
	}

	if strings.Contains(b.FileName, "/") {
		b.FileName = strings.Replace(b.FileName, "/", "-", -1)
	}

	var (
		err           error
		dir           = path.Join(config.Conf.GetString(`function.moefunc.settings.bilibili.localPath`), b.DirName)
		videoURI      = path.Join(dir, utils.StringBuilder(b.FileName, `.video`))
		audioURI      = path.Join(dir, utils.StringBuilder(b.FileName, `.audio`))
		outputFileURI = path.Join(dir, b.FileName)
	)

	Retry := 1
	if config.Conf.GetInt(`worker.aria2.settings.Retry`) > 0 {
		Retry = config.Conf.GetInt(`worker.aria2.settings.Retry`)
	}
	for i := 0; i <= Retry; i++ {
		err = aria2.AwaitHTTPDownload(&aria2.Param{
			DownloadInfoList: []*aria2.DownloadInfo{{
				DownloadType: aria2.DownloadType_HTTP,
				URL:          b.Video,
				Destination:  dir,
				FileName:     utils.StringBuilder(b.FileName, `.video`),
				DownloadOption: &aria2.DownloadOption{
					WithHeader: map[string]string{`Referer`: `https://www.bilibili.com`},
				},
				Token: utils.RandString(6),
			}, {
				DownloadType: aria2.DownloadType_HTTP,
				URL:          b.Audio,
				Destination:  dir,
				FileName:     utils.StringBuilder(b.FileName, `.audio`),
				DownloadOption: &aria2.DownloadOption{
					WithHeader: map[string]string{`Referer`: `https://www.bilibili.com`},
				},
				Token: utils.RandString(6),
			}},
		})
		if err == nil {
			break
		}
		log.Logger.Error(err)
	}

	if err != nil {
		return err
	}
	b.DownloadStatus = 1

	Retry = 1
	if config.Conf.GetInt(`worker.ffmpeg.settings.Retry`) > 0 {
		Retry = config.Conf.GetInt(`worker.ffmpeg.settings.Retry`)
	}

	Options := []ffmpeg.Options{ffmpeg.SetInputVideo(videoURI), ffmpeg.SetInputAudio(audioURI), ffmpeg.SetOutputVideo(outputFileURI)}
	for i := range b.Subtitles {
		if b.Subtitles[i] == nil {
			continue
		}

		Options = append(Options, ffmpeg.WithSubtitle(b.Subtitles[i].Locale, b.Subtitles[i].LocaleText, b.Subtitles[i].SubtitleURL))
	}

	for i := 0; i <= Retry; i++ {
		err = ffmpeg.MergeVideo(Options...)
		if err == nil {
			break
		}
		log.Logger.Error(err)
	}

	if err != nil {
		return err
	}

	Retry = 1
	if config.Conf.GetInt(`worker.fs.settings.Retry`) > 0 {
		Retry = config.Conf.GetInt(`worker.fs.settings.Retry`)
	}

	for i := 0; i <= Retry; i++ {
		err = fileSystemService.DeleteFile(utils.StringBuilder(videoURI, `.aria2`))
		err = fileSystemService.DeleteFile(utils.StringBuilder(audioURI, `.aria2`))
		err = fileSystemService.DeleteFile(videoURI)
		err = fileSystemService.DeleteFile(audioURI)
		err = fileSystemService.DeleteFile(path.Join(dir, `*.srt`))
		if err == nil {
			break
		}
		log.Logger.Error(err)
	}

	if err != nil {
		return err
	}

	remotePath := path.Join(remoteDrive, config.Conf.GetString(`function.moefunc.settings.bilibili.remotePath`), b.DirName)

	for i := 0; i <= Retry; i++ {
		err = fileSystemService.RcloneMove(outputFileURI, remotePath)
		if err == nil {
			break
		}
		log.Logger.Error(err)
	}

	if err != nil {
		return err
	}

	for i := 0; i <= Retry; i++ {
		link, err := fileSystemService.RcloneLink(path.Join(remotePath, b.FileName))
		if err == nil {
			b.RcloneFileLink = link
			break
		}
		log.Logger.Error(err)
	}

	if err != nil {
		return err
	}

	b.DownloadStatus = 2

	return nil
}

func GetEpisodeM3UList(ep *models.BilibiliEpisodeInfo) ([]string, error) {
	if ep == nil {
		return nil, errors.New(`unexpected episode info`)
	}

	s := []string{}
	for i := range ep.Videos {
		for j := range ep.Videos[i].Slices {
			s = append(s, utils.StringBuilder(`#EXTINF:-1,`, ep.Videos[i].Title, ` - `, ep.Videos[i].Slices[j].SubTitle, "\n", ep.Videos[i].Slices[j].Link, "\n"))
		}
	}

	return s, nil
}

func GetArtistM3UList(artist *models.BilibiliArtist) ([]string, error) {
	if artist == nil {
		return nil, errors.New(`unexpected artist info`)
	}

	s := []string{}
	for i := range artist.Videos {
		for j := range artist.Videos[i].Slices {
			s = append(s, utils.StringBuilder(`#EXTINF:-1,`, artist.Videos[i].Title, ` - `, artist.Videos[i].Slices[j].SubTitle, "\n", artist.Videos[i].Slices[j].Link, "\n"))
		}
	}

	return s, nil
}

func GetCollectionM3UList(collection *models.BilibiliArtistCollection) ([]string, error) {
	if collection == nil {
		return nil, errors.New(`unexpected episode info`)
	}

	s := []string{}
	for i := range collection.Videos {
		for j := range collection.Videos[i].Slices {
			s = append(s, utils.StringBuilder(`#EXTINF:-1,`, collection.Videos[i].Title, ` - `, collection.Videos[i].Slices[j].SubTitle, "\n", collection.Videos[i].Slices[j].Link, "\n"))
		}
	}

	return s, nil
}

/*func GetSingleVideoM3UList(video *models.BilibiliVideo) ([]string, error) {
    if video == nil {
        return nil, errors.New(`unexpected episode info`)
    }

    s := []string{}
    for i := range video.Slices {
            s = append(s, utils.StringBuilder(`#EXTINF:-1,`,video.SubTitle, ` - `, video.Slices[i].SubTitle, "\n", video.Slices[i].Link, "\n"))
    }

    return s, nil
}*/

func (b *BilibiliVideo) GetM3U() string {
	if b.DownloadStatus == 2 {
		return utils.StringBuilder(`#EXTINF:-1,`, b.FileName, "\n", b.RcloneFileLink, "\n")
	}

	return ``
}

func UploadBilibiliM3UFile(drive, name string, M3UList []string) (string, error) {
	if strings.Contains(name, "/") {
		name = strings.Replace(name, "/", "-", -1)
	}

	var err error
	sort.Strings(M3UList)

	retry := 1
	if config.Conf.GetInt(`worker.fs.settings.Retry`) > 0 {
		retry = config.Conf.GetInt(`worker.fs.settings.Retry`)
	}

	file := path.Join(config.Conf.GetString(`function.moefunc.settings.bilibili.localPath`), name, utils.StringBuilder(name, `.m3u`))
	M3U := utils.StringBuilder("#EXTM3U\n", utils.StringBuilder(M3UList...))
	for i := 0; i <= retry; i++ {
		err = fileSystemService.CreateEntireFile(utils.StringToByte(M3U), file)
		if err == nil {
			break
		}
		log.Logger.Error(err)
	}

	if err != nil {
		return ``, err
	}

	for i := 0; i <= retry; i++ {
		err = fileSystemService.RcloneMove(file, path.Join(drive, config.Conf.GetString(`function.moefunc.settings.bilibili.remotePath`), name))
		if err == nil {
			break
		}
		log.Logger.Error(err)
	}

	if err != nil {
		return ``, err
	}

	for i := 0; i <= retry; i++ {
		link, err := fileSystemService.RcloneLink(path.Join(drive, config.Conf.GetString(`function.moefunc.settings.bilibili.remotePath`), name, utils.StringBuilder(name, `.m3u`)))
		if err == nil {
			return link, nil
		}
		log.Logger.Error(err)
	}

	return ``, err

}
