package Moefunc

func IsTaskOnGoing(data string) bool {
    VideoList, err := DoBilibiliDataQuery(data, true)
    if err != nil {
        return true
    }
    
    _, ok := PendingTask[VideoList.ID]
    return ok
}
