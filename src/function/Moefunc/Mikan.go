package Moefunc

import (
	"avalon-core/src/config"
	"avalon-core/src/dao"
	"avalon-core/src/dao/models"
	"avalon-core/src/function/aria2"
	"avalon-core/src/function/fileSystemService"
	"avalon-core/src/function/mikan"
	"avalon-core/src/log"
	"avalon-core/src/utils"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func MikanSubscribe(user *models.User, data string) string {
	u, err := url.Parse(data)
	if err != nil {
		log.Logger.Error(err)
		return `订阅失败，请检查链接`
	}

	m := u.Query()
	b, ok := m[`bangumiId`]
	if len(b) == 0 || !ok {
		return `订阅失败，请检查链接`
	}

	s, ok := m[`subgroupid`]
	if len(s) == 0 || !ok {
		return `订阅失败，请检查链接`
	}

	mikanInfo, err := FetchMikanData(b[0], s[0])

	if err != nil {
		return `订阅失败，请检查链接`
	}

	bid, err := strconv.Atoi(b[0])
	if err != nil {
		log.Logger.Error(err)
		return `订阅失败，请检查链接`
	}

	sid, err := strconv.Atoi(s[0])
	if err != nil {
		log.Logger.Error(err)
		return `订阅失败，请检查链接`
	}

	err = dao.CreateNewMikanSubscription(user, bid, sid, mikanInfo.BangumiName)
	if err != nil {
		log.Logger.Error(err)
		return `订阅失败，请检查链接`
	}

	if resp := PrepareResponse(bid, sid, mikanInfo.BangumiName); resp != `` {
		return resp
	}

	return `订阅成功, 更新结束后会向您推送通知`

}

func PrepareResponse(bid, sid int, bangumiName string) string {
	anime, err := dao.FindAnimeSetByIDS(bid, sid)
	if err != nil {
		log.Logger.Error(err)
		return ``
	}

	if anime == nil {
		return ``
	}

	return utils.StringBuilder(`订阅成功, 目前已有资源列表如下:`, "\n", GetMikanUpdateResponse(anime, bangumiName))
}

func UpdateMikanAnimeSet(anime models.MikanAnimeSet) {
	if CheckIsGoing(strconv.Itoa(anime.ID)) {
		return
	}

	mikanData, err := FetchMikanData(strconv.Itoa(anime.AnimeID), strconv.Itoa(anime.SubtitleGroupID))

	if err != nil {
		log.Logger.Error(err)
	}

	CheckBlacklists(anime.BlackListUrls, mikanData)

	defer SendCloseSig(strconv.Itoa(anime.ID))

	needUpdate := IsMikanNeedUpdate(anime.Resources, mikanData)

	if !needUpdate {
		SendSuccessBroadCast(strconv.Itoa(anime.ID), []string{})
		return
	}

	drive, err := GetMikanRcloneDrive()
	if drive == `` || err != nil {
		return
	}

	MikanResources, err := DoMikanDownload(mikanData, drive, anime.ID)
	if err != nil {
		log.Logger.Error(err)
		return
	} else if MikanResources == nil {
		return
	}

	UpdateMikanInfo(&anime, MikanResources)

	M3UList, err := GetMikanM3UList(&anime, mikanData.BangumiName)
	if err != nil {
		log.Logger.Error(err)
		return
	}

	link, err := UploadMikanM3UFile(drive, mikanData.BangumiName, utils.StringBuilder(mikanData.BangumiName, ` - `, strconv.Itoa(anime.SubtitleGroupID)), M3UList)
	if err != nil {
		log.Logger.Error(err)
		return
	}

	anime.M3U8Link = link

	err = dao.SaveMikanAnimeSet(&anime)
	if err != nil {
		log.Logger.Error(err)
		return
	}

	err = MikanSubscriptionSendMiraiBroadCast(utils.StringBuilder(`您订阅的`, mikanData.BangumiName, `已更新:`, "\n", GetMikanUpdateResponse(&anime, mikanData.BangumiName)), anime.ID)
	if err != nil {
		log.Logger.Error(err)
		return
	}

	SendSuccessBroadCast(strconv.Itoa(anime.ID), []string{})
	return

}

func GetMikanM3UList(anime *models.MikanAnimeSet, bangumi string) ([]string, error) {
	if anime == nil {
		return nil, errors.New(`unexpected input anime set`)
	}

	S := []string{}

	for i := range anime.Resources {
		if anime.Resources[i].IsDir {
			continue
		}

		S = append(S, utils.StringBuilder(`#EXTINF:-1,`, bangumi, ` - `, anime.Resources[i].Title, "\n", anime.Resources[i].Link, "\n"))
	}

	sort.Strings(S)
	return S, nil
}

func CheckIsGoing(str string) bool {
	_, ok := PendingTask[str]
	return ok
}

func GetMikanUpdateResponse(anime *models.MikanAnimeSet, bangumi string) string {
	if anime == nil {
		return ``
	}

	S := []string{}
	for i := range anime.Resources {
		S = append(S, utils.StringBuilder(bangumi, ` - `, anime.Resources[i].Title, ":\n", anime.Resources[i].Link, "\n"))
	}

	sort.Strings(S)
	return utils.StringBuilder(utils.StringBuilder(S...), "m3u剧集列表：\n", anime.M3U8Link)
}

func UpdateMikanInfo(anime *models.MikanAnimeSet, resources []*models.MikanResources) {
	if anime == nil {
		return
	}

	for i := range resources {
		if resources[i] == nil {
			continue
		}

		anime.Resources = append(anime.Resources, *resources[i])
	}
}

func GetAndReadToBase64(url string) string {
	for i := 0; i <= 5; i++ {
		str, err := GetTorrent(url)
		if err == nil {
			return str
		}
		log.Logger.Error(err)
	}

	return ``
}

func GetTorrent(u string) (string, error) {
	client := http.Client{}
	req, err := http.NewRequest(`GET`, u, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(respBytes), nil
}

func DoMikanDownload(mikanInfo *mikan.MikanData, drive string, bangumiID int) ([]*models.MikanResources, error) {
	var (
		WG          sync.WaitGroup
		MikanVideos []*models.MikanResources
	)

	if mikanInfo == nil {
		return nil, errors.New(`unexpected mikan info`)
	}

	for index := range mikanInfo.Parts {
		WG.Add(1)
		go func(i int) {

			Video := MikanVideo{
				Title:          mikanInfo.BangumiName,
				SubTitle:       mikanInfo.Parts[i].Title,
				DownloadStatus: 0,
			}

			Retry := 1
			if config.Conf.GetInt(`function.moefunc.settings.mikan.Retry`) > 0 {
				Retry = config.Conf.GetInt(`function.moefunc.settings.mikan.Retry`)
			}

			for j := 0; j <= Retry; j++ {
				u, err := GetTorrent(mikanInfo.Parts[i].URL)
				if err == nil {
					Video.BASE64 = u
					break
				}
				log.Logger.Error(err)
			}

			downloadFinished := CheckAndWaitBTDownload(mikanInfo.Parts[i].Title)
			if !downloadFinished {
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
			}

			Video.GetDir()

			err := Video.DownloadAndUpload(drive, downloadFinished)
			if err != nil {
				log.Logger.Error(err)
			}

			if Video.DownloadStatus == 2 {
				r, err := dao.CreateMikanResource(&models.MikanResources{
					Title:    Video.SubTitle,
					Link:     Video.RcloneLink,
					IsDir:    Video.IsDir,
					BelongTo: bangumiID,
				})

				if err != nil {
					log.Logger.Error(err)
				} else {
					MikanVideos = append(MikanVideos, r)
				}
			}

			WG.Done()
		}(index)
		time.Sleep(10 * time.Second)
	}
	WG.Wait()

	return MikanVideos, nil
}

func CheckAndWaitBTDownload(name string) bool {
	if dao.GetAria2FinishedByName(name) {
		return true
	}

	aria2s, err := dao.FindPendingByName(strings.Replace(name, `/`, `-`, -1))
	if err != nil {
		log.Logger.Error(err)
		return false
	}

	if len(aria2s) == 0 {
		return false
	}

	var GIDS []string

	for i := range aria2s {
		GIDS = append(GIDS, aria2s[i].Aria2Id)
	}

	f, err := aria2.CheckIsDownloadFinished(GIDS)
	if err != nil {
		return false
	} else if f {
		return true
	}

	err = aria2.AwaitGIDDownload(GIDS)

	if err != nil {
		return false
	}

	return true
}

func (m *MikanVideo) GetDir() {
	var (
		title    string
		subTitle string
	)

	if strings.Contains(m.Title, `/`) {
		title = strings.Replace(m.Title, `/`, `-`, -1)
	} else {
		title = m.Title
	}

	if strings.Contains(m.SubTitle, `/`) {
		subTitle = strings.Replace(m.SubTitle, `/`, `-`, -1)
	} else {
		subTitle = m.SubTitle
	}

	m.Dir = path.Join(title, subTitle)
}

func (m *MikanVideo) DownloadAndUpload(remoteDrive string, downloadFinished bool) error {
	var (
		err error
	)

	Retry := 1
	if !downloadFinished {
		if config.Conf.GetInt(`worker.aria2.settings.Retry`) > 0 {
			Retry = config.Conf.GetInt(`worker.aria2.settings.Retry`)
		}
		for i := 0; i <= Retry; i++ {
			err = aria2.AwaitBTDownload(&aria2.Param{
				DownloadInfoList: []*aria2.DownloadInfo{{
					DownloadType: aria2.DownloadType_TORRENT,
					URL:          m.BASE64,
					Destination:  path.Join(config.Conf.GetString(`function.moefunc.settings.mikan.localPath`), m.Dir),
					Token:        utils.RandString(6),
				}},
			}, strings.Replace(m.SubTitle, `/`, `-`, -1))
			if err == nil {
				break
			}
			log.Logger.Error(err)
		}
	}

	if err != nil {
		return err
	}
	m.DownloadStatus = 1

	if err != nil {
		return err
	}

	Retry = 1
	if config.Conf.GetInt(`worker.fs.settings.Retry`) > 0 {
		Retry = config.Conf.GetInt(`worker.fs.settings.Retry`)
	}

	if err != nil {
		return err
	}

	var f []fileSystemService.File
	var filePath string
	for i := 0; i <= Retry; i++ {
		f, err = fileSystemService.ListFile(path.Join(config.Conf.GetString(`function.moefunc.settings.mikan.localPath`), m.Dir))
		if err == nil {
			break
		}
		log.Logger.Error(err)
	}

	if err != nil {
		return err
	}

	for i := range f {
		if strings.Contains(f[i].Name, `.aria2`) || strings.Contains(f[i].Name, `.torrent`) {
			continue
		}
		m.IsDir = f[i].IsDir
		filePath = f[i].Name
	}

	if filePath == `` {
		return errors.New(`file is empty`)
	}

	remotePath := path.Join(remoteDrive, config.Conf.GetString(`function.moefunc.settings.mikan.remotePath`), strings.Replace(m.Title, `/`, `-`, -1), `/`)
	if m.IsDir {
		Retry = 1
		if config.Conf.GetInt(`worker.fs.settings.Retry`) > 0 {
			Retry = config.Conf.GetInt(`worker.fs.settings.Retry`)
		}

		remotePath = path.Join(remotePath, filePath)
		for i := 0; i <= Retry; i++ {
			err = fileSystemService.RcloneMkdir(remotePath)
			if err == nil {
				break
			}
			log.Logger.Error(err)
		}

	}

	Retry = 1
	if config.Conf.GetInt(`worker.fs.settings.Retry`) > 0 {
		Retry = config.Conf.GetInt(`worker.fs.settings.Retry`)
	}

	for i := 0; i <= Retry; i++ {
		err = fileSystemService.RcloneCopy(path.Join(config.Conf.GetString(`function.moefunc.settings.mikan.localPath`), m.Dir, filePath), remotePath)
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

	var linkPath string
	if m.IsDir {
		linkPath = remotePath
	} else {
		linkPath = path.Join(remotePath, filePath)
	}

	for i := 0; i <= Retry; i++ {
		link, err := fileSystemService.RcloneLink(linkPath)
		if err == nil {
			m.RcloneLink = link
			break
		}
		log.Logger.Error(err)
	}

	if err != nil {
		return err
	}

	m.DownloadStatus = 2

	return nil
}

func IsMikanNeedUpdate(resources []models.MikanResources, mikanInfo *mikan.MikanData) bool {
	m := map[string]int{}
	Parts := []mikan.MikanParts{}

	if resources == nil || len(resources) == 0 || mikanInfo == nil {
		return true
	}

	for i := range resources {
		m[resources[i].Title]++
	}

	for i := range mikanInfo.Parts {
		k, _ := m[mikanInfo.Parts[i].Title]
		if k == 0 {
			Parts = append(Parts, mikanInfo.Parts[i])
		}
	}

	mikanInfo.Parts = Parts

	return len(mikanInfo.Parts) != 0
}

func CheckBlacklists(blacklist []models.MikanBlackListUrl, mikanInfo *mikan.MikanData) {
	if blacklist == nil || len(blacklist) == 0 || mikanInfo == nil {
		return
	}

	var parts []mikan.MikanParts

	m := map[string]int{}
	for i := range blacklist {
		m[blacklist[i].URL]++
	}

	for i := range mikanInfo.Parts {
		k, _ := m[mikanInfo.Parts[i].URL]
		if k == 1 {
			continue
		}

		parts = append(parts, mikanInfo.Parts[i])
	}

	mikanInfo.Parts = parts
}

func FetchMikanData(bid, sid string) (*mikan.MikanData, error) {
	var (
		mikanInfo *mikan.MikanData
		err       error
	)

	retry := 1
	if config.Conf.GetInt(`worker.mikan.settings.Retry`) > 0 {
		retry = config.Conf.GetInt(`worker.mikan.settings.Retry`)
	}

	for i := 0; i <= retry; i++ {
		mikanInfo, err = mikan.GetMikanInfo(bid, sid)
		if err == nil && mikanInfo != nil {
			return mikanInfo, nil
		}
		log.Logger.Error(err)
	}

	return nil, err
}

func GetMikanRcloneDrive() (string, error) {
	var (
		drive string
		err   error
		retry = 1
	)
	if config.Conf.GetInt(`worker.fs.settings.Retry`) > 0 {
		retry = config.Conf.GetInt(`worker.fs.settings.Retry`)
	}

	for i := 0; i <= retry; i++ {
		drive, err = fileSystemService.RcloneGetDrive(`mikan`)
		if err == nil && drive != `` {
			return drive, nil
		}
		log.Logger.Error(err)
	}

	return ``, err
}

func UploadMikanM3UFile(drive, name, filename string, M3UList []string) (string, error) {
	name = strings.Replace(name, "/", "-", -1)
	filename = strings.Replace(filename, "/", "-", -1)
	var err error
	sort.Strings(M3UList)

	retry := 1
	if config.Conf.GetInt(`worker.fs.settings.Retry`) > 0 {
		retry = config.Conf.GetInt(`worker.fs.settings.Retry`)
	}

	file := path.Join(config.Conf.GetString(`function.moefunc.settings.mikan.localPath`), name, utils.StringBuilder(filename, `.m3u`))
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
		err = fileSystemService.RcloneMove(file, path.Join(drive, config.Conf.GetString(`function.moefunc.settings.mikan.remotePath`), name))
		if err == nil {
			break
		}
		log.Logger.Error(err)
	}

	if err != nil {
		return ``, err
	}

	for i := 0; i <= retry; i++ {
		link, err := fileSystemService.RcloneLink(path.Join(drive, config.Conf.GetString(`function.moefunc.settings.mikan.remotePath`), name, utils.StringBuilder(filename, `.m3u`)))
		if err == nil {
			return link, nil
		}
		log.Logger.Error(err)
	}

	return ``, err

}

type MikanVideo struct {
	ID             string
	BASE64         string
	Dir            string
	Title          string
	SubTitle       string
	RcloneLink     string
	IsDir          bool
	DownloadStatus int //0 -> not have a task 1 -> have a task but unfinished 2 -> finished
}
