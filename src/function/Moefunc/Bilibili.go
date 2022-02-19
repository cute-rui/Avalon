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
	VideoList, err := FetchBilibiliData(data, true)
	if err != nil {
		return ``
	}

	switch VideoList.Type {
	case bilibili.DataType_Season:
		return BilibiliSubscribe(VideoList, user, data)
	case bilibili.DataType_Video:
		Video := BilibiliSingleVideoDownload(VideoList, data)
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

func BilibiliSingleVideoDownload(VideoList *bilibili.ParsedDownloadInfo, data string) *models.BilibiliVideo {
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
			needUpdate, u := IsSliceNeedUpdate(artist.Videos[i].Slices, VideoList)

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

	BilibiliVideos, err := DoBilibiliSingleVideoDownload(data, drive, updates)
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

func BilibiliUpdateCollectionVideo(VideoList *bilibili.ParsedDownloadInfo, data string) *models.BilibiliVideo {
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
			needUpdate, u := IsSliceNeedUpdate(artist.Videos[i].Slices, VideoList)

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

	BilibiliVideos, err := DoBilibiliSingleVideoDownload(data, drive, updates)
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

func BilibiliCollectionDownload(VideoList *bilibili.ParsedDownloadInfo, data string) string {
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

	if !IsCollectionNeedUpdate(collection, VideoList) {
		return GetCollectionVideoResponse(collection)
	}
	videoInfoMap := map[string]*bilibili.ParsedDownloadInfo{}
	videoListMap := map[string]*models.BilibiliVideo{}
	for i := range VideoList.Parts {
		if v, ok := videoInfoMap[VideoList.Parts[i].VideoID]; ok && v != nil {
			v.Parts = append(v.Parts, VideoList.Parts[i])
		} else {
			v := bilibili.ParsedDownloadInfo{
				Type:   bilibili.DataType_Video,
				ID:     VideoList.Parts[i].VideoID,
				Author: VideoList.Author,
			}
			v.Parts = append(v.Parts, VideoList.Parts[i])
			videoInfoMap[VideoList.Parts[i].VideoID] = &v
		}
	}

	var WG sync.WaitGroup
	for k, v := range videoInfoMap {
		WG.Add(1)
		go func(data string, VideoList *bilibili.ParsedDownloadInfo) {
			Video := BilibiliUpdateCollectionVideo(VideoList, data)
			videoListMap[data] = Video
			WG.Done()
		}(k, v)
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

func BilibiliSubscribe(VideoList *bilibili.ParsedDownloadInfo, user *models.User, data string) string {
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

	needUpdate, u := IsEpisodeNeedUpdate(ep, VideoList)

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

	BilibiliVideos, err := DoBilibiliEpisodeDownload(data, drive, updates)
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
	VideoList, err := FetchBilibiliData(utils.StringBuilder(`https://www.bilibili.com/bangumi/play/ss`, Season), false)
	if err != nil {
		return
	}

	updates := CheckAndWaitTaskFinished(VideoList.ID)

	defer SendCloseSig(VideoList.ID)

	ep, err := dao.FindOrCreateBilibiliEpisodeBySeasonTag(VideoList.ID)
	if err != nil {
		log.Logger.Error(err)
	}

	needUpdate, u := IsEpisodeNeedUpdate(ep, VideoList)

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

	BilibiliVideos, err := DoBilibiliEpisodeDownload(utils.StringBuilder(`https://www.bilibili.com/bangumi/play/ss`, Season), drive, updates)
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

func IsEpisodeNeedUpdate(ep *models.BilibiliEpisodeInfo, VideoList *bilibili.ParsedDownloadInfo) (bool, []string) {
	m := map[string]int{}
	s := []string{}
	if ep == nil || VideoList == nil {
		return false, s
	}

	for i := range ep.Videos {
		for j := range ep.Videos[i].Slices {
			m[utils.StringBuilder(ep.Videos[i].BVID, ep.Videos[i].Slices[j].SubTitle)]++
		}
	}

	for i := range VideoList.Parts {
		k, _ := m[utils.StringBuilder(VideoList.Parts[i].VideoID, VideoList.Parts[i].SubTitle)]
		if k != 1 {
			s = append(s, utils.StringBuilder(VideoList.Parts[i].VideoID, VideoList.Parts[i].SubTitle))
		}
	}

	if len(s) != 0 {
		return true, s
	}

	return false, s
}

func IsCollectionNeedUpdate(collection *models.BilibiliArtistCollection, VideoList *bilibili.ParsedDownloadInfo) bool {
	if collection == nil || VideoList == nil {
		return true
	}

	VideoMap := map[string]int{}
	SliceMap := map[string]int{}

	for i := range collection.Videos {
		VideoMap[collection.Videos[i].BVID] = i
		for j := range collection.Videos[i].Slices {
			SliceMap[utils.StringBuilder(collection.Videos[i].Slices[j].SubTitle, collection.Videos[i].Slices[j].CID)] = j
		}
	}

	for i := range VideoList.Parts {
		if _, ok := VideoMap[VideoList.Parts[i].VideoID]; !ok {
			return true
		} else if _, ok := SliceMap[utils.StringBuilder(VideoList.Parts[i].SubTitle, VideoList.Parts[i].CID)]; !ok {
			return true
		}
	}

	return false
}

func IsSliceNeedUpdate(slices []models.BilibiliSlice, VideoList *bilibili.ParsedDownloadInfo) (bool, []string) {
	m := map[string]int{}
	s := []string{}
	if slices == nil || len(slices) == 0 || VideoList == nil {
		return false, s
	}

	for i := range slices {
		m[utils.StringBuilder(slices[i].CID, slices[i].SubTitle)]++
	}

	for i := range VideoList.Parts {
		k, _ := m[utils.StringBuilder(VideoList.Parts[i].CID, VideoList.Parts[i].SubTitle)]
		if k != 1 {
			s = append(s, utils.StringBuilder(VideoList.Parts[i].CID, VideoList.Parts[i].SubTitle))
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

func DoBilibiliEpisodeDownload(data, drive string, targets []string) ([]*BilibiliVideo, error) {
	var (
		WG             sync.WaitGroup
		BilibiliVideos []*BilibiliVideo
	)

	VideoList, err := FetchBilibiliSeasonData(data, false, targets...)
	log.Logger.Debug(`updating: `, targets)

	if err != nil {
		return nil, err
	}

	for index := range VideoList.Parts {
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

			v, a := GetVideoAndAudioSource(data, VideoList.Parts[i].SubTitle, VideoList.Parts[i].CID)

			Video := BilibiliVideo{
				ID:       VideoList.Parts[i].VideoID,
				CID:      VideoList.Parts[i].CID,
				DirName:  VideoList.CollectionTitle,
				SubTitle: VideoList.Parts[i].SubTitle,
				FileName: utils.StringBuilder(VideoList.CollectionTitle, ` - `, VideoList.Parts[i].SubTitle, ` - `, VideoList.Parts[i].VideoID, ` - `, VideoList.Parts[i].Quality, `.mp4`),
				Audio:    a,
				Video:    v,
			}

			err := Video.DownloadAndUpload(drive)
			if err != nil {
				log.Logger.Error(err)
			}

			if Video.DownloadStatus == 2 {
				BilibiliVideos = append(BilibiliVideos, &Video)
			}

			WG.Done()
		}(index)
		time.Sleep(10 * time.Second)
	}
	WG.Wait()

	return BilibiliVideos, nil
}

func DoBilibiliSingleVideoDownload(data, drive string, targets []string) ([]*BilibiliVideo, error) {
	var (
		WG             sync.WaitGroup
		BilibiliVideos []*BilibiliVideo
	)

	VideoList, err := FetchBilibiliData(data, false, targets...)
	if err != nil {
		return nil, err
	}

	for index := range VideoList.Parts {
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

			v, a := GetVideoAndAudioSource(data, VideoList.Parts[i].SubTitle, VideoList.Parts[i].CID)

			Video := BilibiliVideo{
				ID:       VideoList.Parts[i].VideoID,
				CID:      VideoList.Parts[i].CID,
				DirName:  VideoList.Author,
				Title:    VideoList.Parts[i].Title,
				SubTitle: VideoList.Parts[i].SubTitle,
				FileName: utils.StringBuilder(VideoList.Parts[i].Title, ` - `, VideoList.Parts[i].SubTitle, ` - `, VideoList.Parts[i].VideoID, ` - `, VideoList.Parts[i].Quality, `.mp4`),
				Audio:    a,
				Video:    v,
			}

			err := Video.DownloadAndUpload(drive)
			if err != nil {
				log.Logger.Error(err)
			}

			if Video.DownloadStatus == 2 {
				BilibiliVideos = append(BilibiliVideos, &Video)
			}

			WG.Done()
		}(index)
		time.Sleep(10 * time.Second)
	}
	WG.Wait()

	return BilibiliVideos, nil
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

func GetVideoAndAudioSource(url, title, cid string) (string, string) {
	VideoList, err := FetchBilibiliData(url, false)
	if err != nil {
		return ``, ``
	}

	for i := range VideoList.Parts {
		if VideoList.Parts[i].SubTitle == title && VideoList.Parts[i].CID == cid {
			return VideoList.Parts[i].VideoSource, VideoList.Parts[i].AudioSource
		}
	}

	return ``, ``
}

func FetchBilibiliData(url string, checkCollection bool, targets ...string) (*bilibili.ParsedDownloadInfo, error) {
	var (
		v     *bilibili.ParsedDownloadInfo
		err   error
		retry = 1
	)
	if config.Conf.GetInt(`worker.bilibili.settings.Retry`) > 0 {
		retry = config.Conf.GetInt(`worker.bilibili.settings.Retry`)
	}

	for times := 0; times <= retry; times++ {
		v, err = bilibili.GetDownloadLink(url, checkCollection)
		if err == nil && v != nil {
			if targets == nil {
				return v, nil
			} else {
				var t []bilibili.ParsedPartInfo
				for i := range targets {
					for j := range v.Parts {
						if utils.StringBuilder(v.Parts[j].CID, v.Parts[j].SubTitle) == targets[i] {
							t = append(t, v.Parts[j])
						}
					}
				}
				v.Parts = t
				return v, nil
			}
		}
		log.Logger.Error(err)
	}

	return nil, err
}

func FetchBilibiliSeasonData(url string, checkCollection bool, targets ...string) (*bilibili.ParsedDownloadInfo, error) {
	var (
		v     *bilibili.ParsedDownloadInfo
		err   error
		retry = 1
	)
	if config.Conf.GetInt(`worker.bilibili.settings.Retry`) > 0 {
		retry = config.Conf.GetInt(`worker.bilibili.settings.Retry`)
	}

	for times := 0; times <= retry; times++ {
		v, err = bilibili.GetDownloadLink(url, checkCollection)
		if err == nil && v != nil {
			if targets == nil {
				return v, nil
			} else {
				var t []bilibili.ParsedPartInfo
				for i := range targets {
					for j := range v.Parts {
						if utils.StringBuilder(v.Parts[j].VideoID, v.Parts[j].SubTitle) == targets[i] {
							t = append(t, v.Parts[j])
						}
					}
				}
				v.Parts = t
				return v, nil
			}
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
					if targetsID[k] == utils.StringBuilder(ep.Videos[i].BVID, ep.Videos[i].Slices[j].SubTitle) {
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
		err error
		dir = path.Join(config.Conf.GetString(`function.moefunc.settings.bilibili.localPath`), b.DirName)
		a   = path.Join(dir, utils.StringBuilder(b.FileName, `.video`))
		v   = path.Join(dir, utils.StringBuilder(b.FileName, `.audio`))
		f   = path.Join(dir, b.FileName)
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

	for i := 0; i <= Retry; i++ {
		err = ffmpeg.MergeVideo(v, a, f)
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
		err = fileSystemService.DeleteFile(utils.StringBuilder(v, `.aria2`))
		err = fileSystemService.DeleteFile(utils.StringBuilder(a, `.aria2`))
		err = fileSystemService.DeleteFile(v)
		err = fileSystemService.DeleteFile(a)
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
		err = fileSystemService.RcloneMove(f, remotePath)
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
