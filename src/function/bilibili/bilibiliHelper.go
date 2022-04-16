package bilibili

import (
    "avalon-core/src/config"
    "avalon-core/src/log"
    "avalon-core/src/utils"
    "context"
    "errors"
    "google.golang.org/grpc"
    "strconv"
    "time"
)

var Conn *grpc.ClientConn

func init() {
    c, err := grpc.Dial(config.Conf.GetString(`worker.bilibili.grpc.addr`), grpc.WithInsecure())
    if err != nil {
        log.Logger.Error(err)
    }
    
    Conn = c
}

func DoDownloadQuery(param string, checkCollection bool) (*ParsedQuery, error) {
    client := NewBilibiliClient(Conn)
    
    info, err := client.DoDownloadQuery(context.Background(), &Param{URL: param, CheckCollection: checkCollection})
    
    if err != nil {
        return nil, err
    }
    
    if info.Code != 0 {
        return nil, errors.New(info.GetMsg())
    }
    
    return getDownloadQuery(info)
}

func GetDownloadLink(CID, BVorEPID string, Type DataType, Region Region) ([]*ParsedPartInfo, error) {
    client := NewBilibiliClient(Conn)
    
    info, err := client.GetDownloadInfo(context.Background(), &Param{
        CID:    &CID,
        ID:     &BVorEPID,
        Type:   &Type,
        Region: &Region,
    })
    if err != nil {
        return nil, err
    }
    
    if info.Code != 0 {
        return nil, errors.New(info.GetMsg())
    }
    
    return getDownloadInfo(info)
    
}

func GetInfo(param string) (*Info, error) {
    client := NewBilibiliClient(Conn)
    
    info, err := client.GetInfo(context.Background(), &Param{URL: param})
    if err != nil {
        return nil, err
    }
    
    if info.Code != 0 {
        return nil, errors.New(info.GetMsg())
    }
    
    return info, err
}

func (i *Info) GetString() (string, string) {
    switch i.GetType() {
    case DataType_Video:
        str := utils.StringBuilder(
            i.GetBV(), `/av`, strconv.Itoa(int(i.GetAV())), "\n",
            i.GetTitle(), "\n",
            `作者: `, i.GetAuthor(), "\n",
            `投稿时间: `, time.Unix(i.GetCreateTime(), 0).Format(`2006-01-02 15:04:05`), `/发布时间: `, time.Unix(i.GetPublicTime(), 0).Format(`2006-01-02 15:04:05`), "\n",
            `总时长:`, secondFormat(i.GetDuration()), "\n",
            i.GetDescription(), "\n",
            i.GetAuthor(), `: `, i.GetDynamic(), "\n",
            `https://www.bilibili.com/video/av`, strconv.Itoa(int(i.GetAV())),
        )
        return i.GetPicture(), str
    case DataType_Season:
        var area string
        if i.GetArea() != `` {
            area = utils.StringBuilder(`地区:`, i.GetArea(), "\n")
        }
        str := utils.StringBuilder(
            i.GetTitle(), "\n",
            i.GetEvaluate(), "\n",
            area,
            `状态: `, i.GetSerialStatusDescription(), "\n",
            i.GetShareURL(),
        )
        return i.GetPicture(), str
    case DataType_Collection:
        str := utils.StringBuilder(
            i.GetTitle(), "\n",
            i.GetDescription(), "\n",
            i.GetShareURL(),
        )
        return i.GetPicture(), str
    case DataType_Media:
        var area string
        if i.GetArea() != `` {
            area = utils.StringBuilder(`地区:`, i.GetArea(), "\n")
        }
        str := utils.StringBuilder(
            i.GetTitle(), "\n",
            i.GetEvaluate(), "\n",
            area,
            `状态: `, i.GetSerialStatusDescription(), "\n",
            `分类: `, i.GetMediaType(), "\n",
            i.GetShareURL(),
        )
        return i.GetPicture(), str
    }
    
    return ``, ``
}

func secondFormat(t int64) string {
    var s string
LOOP:
    if t > 3600 {
        s += utils.StringBuilder(strconv.Itoa(int(t/3600)), `小时`)
        t = t % 3600
        goto LOOP
    } else if t > 60 {
        s += utils.StringBuilder(strconv.Itoa(int(t/60)), `分钟`)
        t = t % 60
    }
    return s + utils.StringBuilder(strconv.Itoa(int(t)), `秒`)
}

func getDownloadQuery(info *Query) (*ParsedQuery, error) {
    var s ParsedQuery
    
    s.Type = info.GetType()
    s.Author = info.GetAuthor()
    s.ID = info.GetID()
    s.IsEnd = info.GetIsEnd()
    s.CollectionTitle = info.GetCollectionTitle()
    
    switch info.GetType() {
    case DataType_Video:
        for i := range info.GetDetail() {
            s.Parts = append(s.Parts, ParsedQueryInfo{
                CIDOrEPID: info.GetDetail()[i].GetID(),
                Author:    info.GetDetail()[i].GetAuthor(),
            })
        }
        return &s, nil
    case DataType_Season:
        for i := range info.GetDetail() {
            s.Parts = append(s.Parts, ParsedQueryInfo{
                CIDOrEPID:        info.GetDetail()[i].GetID(),
                BVIDInCollection: info.GetDetail()[i].GetBVID(),
                Region:           info.GetDetail()[i].GetRegion(),
            })
        }
        return &s, nil
    case DataType_Audio:
        for i := range info.GetDetail() {
            s.Parts = append(s.Parts, ParsedQueryInfo{
                CIDOrEPID: info.GetDetail()[i].GetID(),
                Author:    info.GetDetail()[i].GetAuthor(),
            })
        }
        return &s, nil
    case DataType_Article:
        return nil, nil
    case DataType_Collection:
        for i := range info.GetDetail() {
            s.Parts = append(s.Parts, ParsedQueryInfo{
                CIDOrEPID:        info.GetDetail()[i].GetID(),
                BVIDInCollection: info.GetDetail()[i].GetBVID(),
                Author:           info.GetDetail()[i].GetAuthor(),
            })
        }
        return &s, nil
    default:
        return nil, errors.New(`unexpected datatype`)
    }
}

func getDownloadInfo(info *DownloadInfo) ([]*ParsedPartInfo, error) {
    var s []*ParsedPartInfo
    
    switch info.GetType() {
    case DataType_Video:
        for i := range info.GetDetail() {
            s = append(s, &ParsedPartInfo{
                VideoID:       info.GetDetail()[i].GetID(),
                CID:           info.GetDetail()[i].GetCID(),
                Title:         info.GetDetail()[i].GetTitle(),
                SubTitle:      info.GetDetail()[i].GetSubTitle(),
                Quality:       info.GetDetail()[i].GetVideoQuality(),
                VideoSource:   info.GetDetail()[i].GetVideoURL(),
                AudioSource:   info.GetDetail()[i].GetAudioURL(),
                SubtitleInfos: SetSubtitles(info.GetDetail()[i].GetSubtitles()),
            })
        }
        return s, nil
    case DataType_Season:
        for i := range info.GetDetail() {
            s = append(s, &ParsedPartInfo{
                VideoID:       info.GetDetail()[i].GetID(),
                CID:           info.GetDetail()[i].GetCID(),
                SubTitle:      SetSeasonEpisodeTitle(info.GetDetail()[i].GetTitle(), info.GetDetail()[i].GetSubTitle()),
                Quality:       info.GetDetail()[i].GetVideoQuality(),
                VideoSource:   info.GetDetail()[i].GetVideoURL(),
                AudioSource:   info.GetDetail()[i].GetAudioURL(),
                SubtitleInfos: SetSubtitles(info.GetDetail()[i].GetSubtitles()),
            })
        }
        return s, nil
    default:
        return nil, errors.New(`unexpected datatype`)
    }
}

func SetSubtitles(subtitles []*Subtitle) []*SubtitleInfo {
    var SubtitleInfos []*SubtitleInfo
    for i := range subtitles {
        SubtitleInfos = append(SubtitleInfos, &SubtitleInfo{
            Locale:      subtitles[i].GetLocale(),
            LocaleText:  subtitles[i].GetLocaleText(),
            SubtitleURL: subtitles[i].GetSubtitleUrl(),
        })
    }
    
    return SubtitleInfos
}

func SetSeasonEpisodeTitle(title, subtitle string) string {
    if subtitle != `` {
        title = utils.StringBuilder(subtitle, ` - `, title)
    }
    
    if title == `` {
        title = subtitle
    }
    
    return title
}

type ParsedQuery struct {
    Type DataType
    //Video Season Collection ID
    ID              string
    CollectionTitle string
    Author          string
    IsEnd           bool
    Parts           []ParsedQueryInfo
}

type ParsedQueryInfo struct {
    Author           string
    CIDOrEPID        string
    BVIDInCollection string
    Region           Region
}

type ParsedPartInfo struct {
    VideoID       string //BVID
    CID           string
    Title         string
    SubTitle      string
    Quality       string
    VideoSource   string
    AudioSource   string
    SubtitleInfos []*SubtitleInfo
}

type SubtitleInfo struct {
    Locale      string
    LocaleText  string
    SubtitleURL string
}

type DetailInfo struct {
    Type       int
    Picture    string
    BV         string
    Aid        int64
    Title      string
    PubDate    int64
    CreateTime int64
    Duration   string
    Author     string
    Dynamic    string
    //Media
    TypeName string
    //Season or Episode
    Area     string
    Status   string
    ShareURL string
    Evaluate string
}
