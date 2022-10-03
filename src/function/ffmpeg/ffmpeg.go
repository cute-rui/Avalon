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

func MergeVideo(options ...Options) error {
	client := NewFFMPEGClient(Conn)

	param := Param{}

	for i := range options {
		options[i](&param)
	}

	info, err := client.MergeVideo(context.Background(), &param)

	if err != nil {
		return err
	}

	if info.Code != 0 {
		return errors.New(info.Msg)
	}

	return nil
}

type Options func(param *Param)

func SetInputVideo(value string) Options {
	return func(param *Param) {
		if param == nil {
			return
		}

		param.InputVideo = value
	}
}

func SetOutputVideo(value string) Options {
	return func(param *Param) {
		if param == nil {
			return
		}

		param.OutputVideo = value
	}
}

func SetInputAudio(value string) Options {
	return func(param *Param) {
		if param == nil {
			return
		}

		param.InputAudio = value
	}
}

func WithSubtitle(locale, localeText, Url string) Options {
	return func(param *Param) {
		if param == nil {
			return
		}

		param.Subtitles = append(param.GetSubtitles(), &Subtitle{
			Locale:      locale,
			LocaleText:  localeText,
			SubtitleUrl: Url,
		})
	}
}
