package test

/*func Testcron1() {
    log.Println(`testcron on going`)
    s := gocron.NewScheduler(time.UTC)
    location, err := time.LoadLocation("Asia/Shanghai")
    if err != nil {
        log.Println(err)
        return
    }
    s.ChangeLocation(location)
    _, err = s.Every(1).Second().Do(log.Println, "114514")
    if err != nil {
        log.Println(err)
        return
    }
    s.StartBlocking()
}

func Test1(pipes *utils.TransmitterPipes) {
    log.Println(`test1 on going`)
    for {
        select {
        case c := <-*pipes.In:
            log.Println(c.Event)
            *pipes.Out <- utils.Transmit{Event: `Test1CallTest2`, Destination: `test2`}
            *pipes.Out <- utils.Transmit{Event: `Test1CallALL`, Destination: ``}
        }
    }
}

func Test2(pipes *utils.TransmitterPipes) {
    log.Println(`test2 on going`)
    for {
        select {
        case c := <-*pipes.In:
            log.Println(c.Event)
            *pipes.Out <- utils.Transmit{Event: `Test2CallTest1`, Destination: `test1`}
        }
    }
}*/
