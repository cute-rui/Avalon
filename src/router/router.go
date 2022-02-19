package router

import (
	"avalon-core/src/controller"
	"avalon-core/src/function/Moefunc"
	"avalon-core/src/function/bilibili"
	"avalon-core/src/function/ffmpeg"
	"avalon-core/src/function/fileSystemService"
	"avalon-core/src/function/mikan"
	"avalon-core/src/function/mirai"
)

func Start() {
	r := controller.Default()
	r.MountTransmitters(`mirai`, mirai.MiraiListener)
	r.MountTransmitters(`aria2`, mikan.MikanListener)
	r.MountTransmitters(`bilibili`, bilibili.BilibiliListener)
	r.MountTransmitters(`mikan`, mikan.MikanListener)
	r.MountTransmitters(`fs`, fileSystemService.FSListener)
	r.MountTransmitters(`ffmpeg`, ffmpeg.FFMPEGListener)
	r.MountTransmitters(`moefunc`, Moefunc.MoeFuncListener)
	r.MountCrons(`AutoUpdate`, Moefunc.AnimeUpdater)
	r.Load()
}
