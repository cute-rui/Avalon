package Moefunc

import (
	"avalon-core/src/config"
	"avalon-core/src/dao"
	"avalon-core/src/function/bilibili"
	"avalon-core/src/log"
	"avalon-core/src/utils"
	"github.com/go-co-op/gocron"
	jsoniter "github.com/json-iterator/go"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var bilibiliAnimeTimeLine map[string]string
var TimeLineWriteLock sync.Mutex

func init() {
	bilibiliAnimeTimeLine = make(map[string]string)
}

func AnimeUpdater() {
	s := gocron.NewScheduler(time.UTC)
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Logger.Error(err)
		return
	}
	s.ChangeLocation(location)
	_, err = s.Every(1).Day().At("00:01").StartImmediately().Do(UpdateBilibiliData, s)
	_, err = s.Every(1).Hour().StartImmediately().Do(UpdateMikanData)
	if err != nil {
		log.Logger.Error(err)
		return
	}
	s.StartBlocking()
}

func UpdateBilibiliData(scheduler *gocron.Scheduler) {
	bilibiliAnimeTimeLine = make(map[string]string)

	var data AnimeTimeLine
	err := fetchAnimeTimeLineData(&data)
	if err != nil {
		log.Logger.Error(err)
	}

	if data.Code != 0 {
		log.Logger.Error(`anime timeline fetch failed`)
	}

	episodes, err := dao.ListAllBilibiliSubscriptions()
	if err != nil {
		log.Logger.Error(err)
	}

	for i := range episodes {
		if CheckIsEnd(episodes[i].SeasonTag) {
			err = dao.DeleteSubscription(&episodes[i])
			if err != nil {
				log.Logger.Error(err)
			}
		}
	}

	episodes, err = dao.ListAllBilibiliSubscriptions()
	if err != nil {
		log.Logger.Error(err)
	}

	k := GetLastUpdate()

	for i := range data.Result {
		if data.Result[i].IsToday != 1 {
			continue
		}
		for l := 0; l <= k; l++ {
			if i-l < 0 {
				break
			}
			for j := range data.Result[i-l].Seasons {
				if data.Result[i-l].Seasons[j].Delay != 0 {
					continue
				}

				if data.Result[i-l].Seasons[j].PubTs < time.Now().Unix() {
					TimeLineWriteLock.Lock()
					bilibiliAnimeTimeLine[strconv.Itoa(data.Result[i-l].Seasons[j].SeasonId)] = ""
					TimeLineWriteLock.Unlock()
				} else {
					TimeLineWriteLock.Lock()
					bilibiliAnimeTimeLine[strconv.Itoa(data.Result[i-l].Seasons[j].SeasonId)] = data.Result[i-l].Seasons[j].PubTime
					TimeLineWriteLock.Unlock()
				}
			}
		}
	}

	var WG sync.WaitGroup
	for i := range episodes {
		if _, ok := bilibiliAnimeTimeLine[episodes[i].SeasonTag]; ok {
			WG.Add(1)
			go func(index int) {
				bilibiliIntervalTaskTimer(scheduler, episodes[index].SeasonTag)
				WG.Done()
			}(i)
		}
	}

	WG.Wait()
	SetLastUpdate()
}

func UpdateMikanData() {
	animeSets, err := dao.ListAllMikanSubscriptions()
	if err != nil {
		log.Logger.Error(err)
	}

	for i := range animeSets {
		go func(index int) {
			UpdateMikanAnimeSet(animeSets[index])
		}(i)
	}
}

func bilibiliIntervalTaskTimer(s *gocron.Scheduler, id string) {
	if bilibiliAnimeTimeLine[id] == "" {
		BilibiliMediaSubscriptionController(id)
		return
	}

	j, err := s.Every(1).Day().At(bilibiliAnimeTimeLine[id]).WaitForSchedule().Do(BilibiliMediaSubscriptionController, id)
	if err != nil {
		log.Logger.Error(err)
		return
	}
	j.LimitRunsTo(1)
}

func BilibiliMediaSubscriptionController(id string) {
	var loop_index int
LOOP:
	log.Logger.Debug(`BILIBILI MEDIA SUBSCRIPTION CONTROLLER IS WORKING: `, id)
	if loop_index > 5 {
		log.Logger.Error(`error on sub controller`)
		return
	}

	var data AnimeTimeLine
	err := fetchAnimeTimeLineData(&data)
	if err != nil {
		log.Logger.Error(err)
	}

	if data.Code != 0 {
		log.Logger.Error(`anime timeline fetch failed`)
	}

	k := GetLastUpdate()

	var published bool
	for i := range data.Result {
		if data.Result[i].IsToday != 1 {
			continue
		}
		for l := 0; l <= k; l++ {
			if i-l < 0 {
				break
			}
			for j := range data.Result[i-l].Seasons {
				if strconv.Itoa(data.Result[i-l].Seasons[j].SeasonId) != id {
					continue
				}

				if data.Result[i-l].Seasons[j].IsPublished == 1 {
					published = true
				}
				break
			}
		}
		break
	}

	if !published {
		loop_index++
		log.Logger.Info(utils.StringBuilder(id, `not published, sleeping`))
		time.Sleep(5 * time.Minute)
		goto LOOP
	}

	BilibiliSubscribeUpdate(id)
}

func GetLastUpdate() int {
	settings, err := dao.GetSettings()
	if err != nil {
		log.Logger.Error(err)
		return 0
	}

	oldt := settings.BilibiliLastUpdate.Unix()

	t := time.Now().Unix()

	t = (t - oldt) % 604800
	return time.Unix(t, 0).Day() - 1
}

func SetLastUpdate() {
	err := dao.SetBilibiliLastUpdate(time.Now())
	if err != nil {
		log.Logger.Error(err)
	}
}

func CheckIsEnd(ss string) bool {
	var VideoList *bilibili.ParsedDownloadInfo
	var err error
	retry := 1
	if config.Conf.GetInt(`worker.bilibili.settings.Retry`) > 0 {
		retry = config.Conf.GetInt(`worker.bilibili.settings.Retry`)
	}

	for i := 0; i <= retry; i++ {
		v, err := bilibili.GetDownloadLink(utils.StringBuilder(`https://www.bilibili.com/bangumi/play/ss`, ss), false)
		if err == nil && v != nil {
			VideoList = v
			break
		}
		log.Logger.Error(err)
	}

	if err != nil {
		return false
	}

	return VideoList.IsEnd
}

func fetchAnimeTimeLineData(result interface{}) error {
	client := http.Client{}

	req, err := http.NewRequest(`GET`, `https://bangumi.bilibili.com/web_api/timeline_global`, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = jsoniter.Unmarshal(respBytes, result)
	if err != nil {
		return err
	}

	return nil
}

type AnimeTimeLine struct {
	Code   int `json:"code"`
	Result []struct {
		Date      string `json:"date"`
		DateTs    int    `json:"date_ts"`
		DayOfWeek int    `json:"day_of_week"`
		IsToday   int    `json:"is_today"`
		Seasons   []struct {
			Cover        string `json:"cover"`
			Delay        int    `json:"delay"`
			EpId         int    `json:"ep_id"`
			Favorites    int    `json:"favorites"`
			Follow       int    `json:"follow"`
			IsPublished  int    `json:"is_published"`
			PubIndex     string `json:"pub_index,omitempty"`
			PubTime      string `json:"pub_time"`
			PubTs        int64  `json:"pub_ts"`
			SeasonId     int    `json:"season_id"`
			SeasonStatus int    `json:"season_status"`
			SquareCover  string `json:"square_cover"`
			Title        string `json:"title"`
			Url          string `json:"url"`
			DelayId      int    `json:"delay_id,omitempty"`
			DelayIndex   string `json:"delay_index,omitempty"`
			DelayReason  string `json:"delay_reason,omitempty"`
		} `json:"seasons"`
	} `json:"result,omitempty"`
}
