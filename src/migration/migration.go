package migration

import (
	"avalon-core/src/config"
	"avalon-core/src/dao"
	"avalon-core/src/dao/models"
	"avalon-core/src/function/Moefunc"
	"avalon-core/src/function/fileSystemService"
	"avalon-core/src/log"
	"avalon-core/src/utils"
	"gorm.io/gorm/clause"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func DeleteReturn() {
	db := dao.GetMySQLInstance().Database

	var resources []models.MikanResources

	db.Find(&resources)

	for i := range resources {
		resources[i].Link = strings.TrimSuffix(resources[i].Link, "\n")
	}

	db.Save(&resources)
}

func UpdateMikanAnimeM3U() {
	Series, err := dao.ListAllAnimeSeries()
	if err != nil {
		log.Logger.Error(err)
		return
	}
	var WG sync.WaitGroup

	for i := range Series {
		for j := range Series[i].AnimeSets {
			WG.Add(1)
			go func(i, j int) {
				list, err := Moefunc.GetMikanM3UList(&Series[i].AnimeSets[j], Series[i].Name)
				if err != nil {
					log.Logger.Error(err)
				}

				drive, err := Moefunc.GetMikanRcloneDrive()
				if drive == `` || err != nil {
					log.Logger.Error(err)
				}

				link, err := UploadMikanM3UFile(drive, Series[i].Name, utils.StringBuilder(Series[i].Name, ` - `, strconv.Itoa(Series[i].AnimeSets[j].SubtitleGroupID)), list)
				if err != nil {
					log.Logger.Error(err)
				}

				Series[i].AnimeSets[j].M3U8Link = link

				err = dao.SaveMikanAnimeSet(&Series[i].AnimeSets[j])
				if err != nil {
					log.Logger.Error(err)
				}
				WG.Done()
			}(i, j)
			time.Sleep(3 * time.Second)
		}
	}
	WG.Wait()
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

	file := path.Join(config.Conf.GetString(`function.moefunc.settings.mikan.localPath`), `migration`, utils.StringBuilder(filename, `.m3u`))
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

func UpdateBilibiliEpisodeM3U() {
	mysql := dao.GetMySQLInstance().Database

	var episodes []models.BilibiliEpisodeInfo
	err := mysql.Preload(clause.Associations).Preload("Videos.Slices").Find(&episodes).Error

	if err != nil {
		log.Logger.Error(err)
		return
	}

	var WG sync.WaitGroup

	for i := range episodes {
		if episodes[i].SeasonTag != "39487" && episodes[i].SeasonTag != "39463" && episodes[i].SeasonTag != "39468" && episodes[i].SeasonTag != "5978" {
			continue
		}

		WG.Add(1)
		go func(i int) {
			drive, err := Moefunc.GetBilibiliRcloneDrive()
			if drive == `` || err != nil {
				log.Logger.Error(err)
			}

			list, err := Moefunc.GetEpisodeM3UList(&episodes[i])
			if err != nil {
				log.Logger.Error(err)
			}

			if len(episodes[i].Videos) < 1 {
				log.Logger.Debug(`video less than 1`, episodes[i].SeasonTag)
				WG.Done()
				return
			}

			link, err := UploadBilibiliM3UFile(drive, episodes[i].Videos[0].Title, list)
			if err != nil {
				log.Logger.Error(err)
			}

			episodes[i].M3U8Link = link

			err = dao.SaveEpisode(&episodes[i])
			if err != nil {
				log.Logger.Error(err)
			}
			WG.Done()
		}(i)

		time.Sleep(3 * time.Second)
	}

	WG.Wait()
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

	file := path.Join(config.Conf.GetString(`function.moefunc.settings.bilibili.localPath`), `migration`, utils.StringBuilder(name, `.m3u`))
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

func UpdateBilibiliArtistM3U() {
	mysql := dao.GetMySQLInstance().Database

	var artists []models.BilibiliArtist
	err := mysql.Preload(clause.Associations).Preload("Videos.Slices").Find(&artists).Error
	if err != nil {
		log.Logger.Error(err)
		return
	}

	var WG sync.WaitGroup

	for i := range artists {
		WG.Add(1)
		go func(i int) {
			drive, err := Moefunc.GetBilibiliRcloneDrive()
			if drive == `` || err != nil {
				log.Logger.Error(err)
			}

			list, err := Moefunc.GetArtistM3UList(&artists[i])
			if err != nil {
				log.Logger.Error(err)
			}

			link, err := UploadBilibiliM3UFile(drive, artists[i].Artist, list)
			if err != nil {
				log.Logger.Error(err)
			}

			artists[i].M3U8Link = link

			err = dao.SaveArtist(&artists[i])
			if err != nil {
				log.Logger.Error(err)
			}

			WG.Done()
		}(i)
		time.Sleep(3 * time.Second)
	}

	WG.Wait()
}
