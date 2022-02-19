package main

import (
	"avalon-core/src/log"
	"net/http"
	_ "net/http/pprof"
)

func initPprofMonitor() error {

	var err error

	go func() {
		err = http.ListenAndServe(":7777", nil)
		if err != nil {
			log.Logger.Error("funcRetErr=http.ListenAndServe||err=%s", err.Error())
		}
	}()

	return err
}

func migrate() {
	//migration.DeleteReturn()
	//migration.UpdateMikanAnimeM3U()
	//migration.UpdateBilibiliArtistM3U()
	//migration.UpdateBilibiliEpisodeM3U()
}
