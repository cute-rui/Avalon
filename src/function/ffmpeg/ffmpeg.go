package ffmpeg

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
	c, err := grpc.Dial(config.Conf.GetString(`worker.ffmpeg.grpc.addr`), grpc.WithInsecure())
	if err != nil {
		log.Logger.Error(err)
	}

	Conn = c
}

func FFMPEGListener(context.Context, *utils.TransmitterPipes) {

}

func MergeVideo(v, a, output string) error {
	client := NewFFMPEGClient(Conn)

	info, err := client.MergeVideo(context.Background(), &Param{
		InputVideo:  v,
		InputAudio:  a,
		OutputVideo: output,
	})

	if err != nil {
		return err
	}

	if info.Code != 0 {
		return errors.New(info.Msg)
	}

	return nil
}
