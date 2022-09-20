package config

import (
	"avalon-core/src/log"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"os"
	"strings"
	"time"
)

var Conf = viper.New()

var WriteConfig = flag.Bool("config", true, "If write config")

func confInit() {
	Conf.SetConfigType(`toml`)
	Conf.SetConfigName(`avalon-core`)
	Conf.AddConfigPath(`./soft/avalon/config/`)

	Conf.SetDefault(`database.user`, ``)
	Conf.SetDefault(`database.pass`, ``)
	Conf.SetDefault(`database.host`, ``)
	Conf.SetDefault(`database.port`, ``)
	Conf.SetDefault(`database.name`, `avalon`)
	Conf.SetDefault(`database.logMode`, true)

	//Worker
	Conf.SetDefault(`worker.bilibili.grpc.addr`, `localhost:6231`)
	Conf.SetDefault(`worker.bilibili.grpc.auth.tls`, false)
	Conf.SetDefault(`worker.bilibili.grpc.auth.tlsPath`, ``)
	Conf.SetDefault(`worker.bilibili.settings.Retry`, 5)

	Conf.SetDefault(`worker.mikan.grpc.addr`, `localhost:6232`)
	Conf.SetDefault(`worker.mikan.grpc.auth.tls`, false)
	Conf.SetDefault(`worker.mikan.grpc.auth.tlsPath`, ``)
	Conf.SetDefault(`worker.mikan.settings.Retry`, 5)

	Conf.SetDefault(`worker.fs.grpc.addr`, `localhost:6222`)
	Conf.SetDefault(`worker.fs.grpc.auth.tls`, false)
	Conf.SetDefault(`worker.fs.grpc.auth.tlsPath`, ``)
	Conf.SetDefault(`worker.fs.settings.rclone.maxFilled`, 30)
	Conf.SetDefault(`worker.fs.settings.Retry`, 5)

	Conf.SetDefault(`worker.ffmpeg.grpc.addr`, `localhost:6221`)
	Conf.SetDefault(`worker.ffmpeg.grpc.auth.tls`, false)
	Conf.SetDefault(`worker.ffmpeg.grpc.auth.tlsPath`, ``)
	Conf.SetDefault(`worker.ffmpeg.settings.Retry`, 5)

	Conf.SetDefault(`worker.mirai.grpc.addr`, `localhost:6212`)
	Conf.SetDefault(`worker.mirai.grpc.auth.tls`, false)
	Conf.SetDefault(`worker.mirai.account.channel`, ``)
	Conf.SetDefault(`worker.mirai.account.verifyKey`, ``)
	Conf.SetDefault(`worker.mirai.account.qq`, 0)
	Conf.SetDefault(`worker.mirai.settings.Retry`, 5)

	Conf.SetDefault(`worker.aria2.grpc.addr`, `localhost:6211`)
	Conf.SetDefault(`worker.aria2.grpc.auth.tls`, false)
	Conf.SetDefault(`worker.aria2.grpc.auth.tlsPath`, ``)
	Conf.SetDefault(`worker.aria2.settings.maxDownloadTasks`, 5)
	Conf.SetDefault(`worker.aria2.settings.Retry`, 5)

	//Logger
	Conf.SetDefault(`logger.isProduction`, false)

	Conf.SetDefault(`logger.error.enable`, true)
	Conf.SetDefault(`logger.error.filePath`, `./log/error.log`)
	Conf.SetDefault(`logger.error.maxSize`, 1)
	Conf.SetDefault(`logger.error.maxBackups`, 5)
	Conf.SetDefault(`logger.error.maxAge`, 30)
	Conf.SetDefault(`logger.error.compress`, false)

	Conf.SetDefault(`logger.info.enable`, true)
	Conf.SetDefault(`logger.info.filePath`, `./log/info.log`)
	Conf.SetDefault(`logger.info.maxSize`, 2)
	Conf.SetDefault(`logger.info.maxBackups`, 100)
	Conf.SetDefault(`logger.info.maxAge`, 30)
	Conf.SetDefault(`logger.info.compress`, false)

	Conf.SetDefault(`logger.debug.enable`, true)
	Conf.SetDefault(`logger.debug.filePath`, `./log/debug.log`)
	Conf.SetDefault(`logger.debug.maxSize`, 2)
	Conf.SetDefault(`logger.debug.maxBackups`, 100)
	Conf.SetDefault(`logger.debug.maxAge`, 30)
	Conf.SetDefault(`logger.debug.compress`, true)

	Conf.SetDefault(`function.moefunc.settings.bilibili.lastupdate`, time.Now())
	Conf.SetDefault(`function.moefunc.settings.bilibili.localPath`, ``)
	Conf.SetDefault(`function.moefunc.settings.bilibili.remotePath`, ``)
	Conf.SetDefault(`function.moefunc.settings.mikan.Retry`, 5)
	Conf.SetDefault(`function.moefunc.settings.mikan.localPath`, ``)
	Conf.SetDefault(`function.moefunc.settings.mikan.remotePath`, ``)

	replacer := strings.NewReplacer(`.`, `_`)
	Conf.SetEnvKeyReplacer(replacer)
	err := Conf.ReadInConfig()
	initLogger()
	if *WriteConfig {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			_, err := os.Create(`./soft/avalon/config/avalon-core.toml`)
			if err != nil {
				log.Logger.Error(err)
				return
			}
		}
		err = Conf.WriteConfig()
		if err != nil {
			log.Logger.Error(err)
			return
		}
	}
	Conf.WatchConfig()
	Conf.OnConfigChange(func(in fsnotify.Event) {
		err := Conf.ReadInConfig()
		if err != nil {
			log.Logger.Error(err)
		}
	})
}

func initLogger() {
	log.LogConf.IsProduction = Conf.GetBool(`logger.isProduction`)
	log.LogConf.ErrorFile = log.FileConfig{
		Enabled:    Conf.GetBool(`logger.error.enable`),
		Filename:   Conf.GetString(`logger.error.filePath`),
		MaxSize:    Conf.GetInt(`logger.error.maxSize`),
		MaxBackups: Conf.GetInt(`logger.error.maxBackups`),
		MaxAge:     Conf.GetInt(`logger.error.maxAge`),
		Compress:   Conf.GetBool(`logger.error.compress`),
	}

	log.LogConf.InfoFile = log.FileConfig{
		Enabled:    Conf.GetBool(`logger.info.enable`),
		Filename:   Conf.GetString(`logger.info.filePath`),
		MaxSize:    Conf.GetInt(`logger.info.maxSize`),
		MaxBackups: Conf.GetInt(`logger.info.maxBackups`),
		MaxAge:     Conf.GetInt(`logger.info.maxAge`),
		Compress:   Conf.GetBool(`logger.info.compress`),
	}

	log.LogConf.DebugFile = log.FileConfig{
		Enabled:    Conf.GetBool(`logger.debug.enable`),
		Filename:   Conf.GetString(`logger.debug.filePath`),
		MaxSize:    Conf.GetInt(`logger.debug.maxSize`),
		MaxBackups: Conf.GetInt(`logger.debug.maxBackups`),
		MaxAge:     Conf.GetInt(`logger.debug.maxAge`),
		Compress:   Conf.GetBool(`logger.debug.compress`),
	}

	log.InitLogger()
}

func init() {
	confInit()
}
