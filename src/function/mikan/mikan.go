package mikan

import (
	"avalon-core/src/config"
	"avalon-core/src/log"
	"avalon-core/src/utils"
	"context"
	"errors"
	"google.golang.org/grpc"
)

var Conn *grpc.ClientConn

func init() {
	c, err := grpc.Dial(config.Conf.GetString(`worker.mikan.grpc.addr`), grpc.WithInsecure())
	if err != nil {
		log.Logger.Error(err)
	}

	Conn = c
}

func MikanListener(context.Context, *utils.TransmitterPipes) {

}

func GetMikanInfoByURL(url string) {

}

func GetMikanInfo(bangumi string, subgroup string) (*MikanData, error) {
	client := NewMikanClient(Conn)

	info, err := client.GetInfo(context.Background(), &Param{
		Bangumi:  bangumi,
		Subgroup: &subgroup,
	})

	if err != nil {
		return nil, err
	}

	if info.GetCode() != 0 {
		return nil, errors.New(info.GetMsg())
	}

	mikanData := MikanData{
		BangumiName: info.GetBangumiName(),
	}

	for i := range info.GetData() {
		mikanData.Parts = append(mikanData.Parts, MikanParts{
			Title: info.GetData()[i].GetTitle(),
			URL:   info.GetData()[i].GetURL(),
			Type:  int(info.GetData()[i].GetDataType()),
		})
	}

	return &mikanData, nil
}

type MikanData struct {
	BangumiName string
	Parts       []MikanParts
}

type MikanParts struct {
	Title string
	URL   string
	Type  int
}
