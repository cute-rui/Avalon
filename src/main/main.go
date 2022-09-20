package main

import (
	"avalon-core/src/dao"
	"avalon-core/src/router"
	"flag"
)

func main() {
	flag.Parse()
	
	initPprofMonitor()
	dao.InitDatabase()

	migrate()
	router.Start()
}
