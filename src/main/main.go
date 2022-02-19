package main

import (
	"avalon-core/src/dao"
	"avalon-core/src/router"
)

func main() {
	initPprofMonitor()
	dao.InitDatabase()

	migrate()
	router.Start()
}
