package aria2

import (
	"avalon-core/src/config"
	"avalon-core/src/dao"
	"avalon-core/src/log"
	"avalon-core/src/utils"
	"context"
	"errors"
	"google.golang.org/grpc"
	"io"
)

var Conn *grpc.ClientConn

func init() {
	c, err := grpc.Dial(config.Conf.GetString(`worker.aria2.grpc.addr`), grpc.WithInsecure())
	if err != nil {
		log.Logger.Error(err)
	}

	Conn = c
}

func AwaitHTTPDownload(Param *Param) error {
	client := NewAria2AgentClient(Conn)

	stream, err := client.AwaitDownload(context.Background(), Param)
	if err != nil {
		return err
	}

	GIDS := []string{}
	defer CleanGIDPointer(&GIDS)

	for {
		data, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		databaseAccessErrors := map[string]error{}
		switch data.GetCode() {
		case 400:
			return errors.New(data.Msg)
		case 0:
			if data.GetFileInfo() != nil {
				if len(data.GetFileInfo()) == 0 {
					return errors.New(`Unexpected FileInfo`)
				}

				for _, v := range data.GetFileInfo() {
					if _, ok := databaseAccessErrors[v.GetGID()]; ok {
						continue
					}
					if v.GetIsFinished() && v.GetGID() != `` {
						err := dao.DeletePendingByGID(v.GetGID())
						if err != nil {
							databaseAccessErrors[v.GetGID()] = err
						}
					}
				}
			} else {
				return errors.New(`Unexpected FileInfo`)
			}

			if len(databaseAccessErrors) != 0 {
				for k, v := range databaseAccessErrors {
					log.Logger.Error(v, `: `, k)
				}
				return errors.New(`database operation failed`)
			}

			return nil
		case 1:
			if data.GetFileInfo() != nil {
				if len(data.GetFileInfo()) == 0 {
					return errors.New(`Unexpected FileInfo`)
				}

				for k, v := range data.GetFileInfo() {
					if !v.GetIsFinished() && v.GetGID() != `` {
						GIDS = append(GIDS, v.GetGID())
						var name string
						for i := range Param.DownloadInfoList {
							if Param.DownloadInfoList[i].Token != k {
								continue
							}
							name = Param.DownloadInfoList[i].FileName
						}
						_, _, err := dao.FindOrCreatePending(v.GetGID(), utils.StringBuilder(name, `HTTP`))
						if err != nil {
							databaseAccessErrors[v.GetGID()] = err
						}
					}
				}
			} else {
				return errors.New(`Unexpected FileInfo`)
			}
		}
	}
	return errors.New(`time out`)
}

func AwaitBTDownload(Param *Param, Name string) error {
	client := NewAria2AgentClient(Conn)

	stream, err := client.AwaitDownload(context.Background(), Param)
	if err != nil {
		return err
	}

	GIDS := []string{}
	defer CleanGIDPointer(&GIDS)

	for {
		data, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		databaseAccessErrors := map[string]error{}
		switch data.GetCode() {
		case 400:
			return errors.New(data.Msg)
		case 0:
			if data.GetFileInfo() != nil {
				if len(data.GetFileInfo()) == 0 {
					return errors.New(`Unexpected FileInfo`)
				}

				for _, v := range data.GetFileInfo() {
					if _, ok := databaseAccessErrors[v.GetGID()]; ok {
						continue
					}
					if v.GetIsFinished() && v.GetGID() != `` {
						err := dao.DeletePendingByGID(v.GetGID())
						if err != nil {
							databaseAccessErrors[v.GetGID()] = err
						}
					}
				}
			} else {
				return errors.New(`Unexpected FileInfo`)
			}

			if len(databaseAccessErrors) != 0 {
				for k, v := range databaseAccessErrors {
					log.Logger.Error(v, `: `, k)
				}
				return errors.New(`database operation failed`)
			}

			return nil
		case 1:
			if data.GetFileInfo() != nil {
				if len(data.GetFileInfo()) == 0 {
					return errors.New(`Unexpected FileInfo`)
				}

				for k, v := range data.GetFileInfo() {
					if !v.GetIsFinished() && v.GetGID() != `` {
						GIDS = append(GIDS, v.GetGID())
						for i := range Param.DownloadInfoList {
							if Param.DownloadInfoList[i].Token != k {
								continue
							}
						}
						_, _, err := dao.FindOrCreatePending(v.GetGID(), utils.StringBuilder(Name, `BT`))
						if err != nil {
							databaseAccessErrors[v.GetGID()] = err
						}
					}
				}
			} else {
				return errors.New(`Unexpected FileInfo`)
			}
		}
	}
	return errors.New(`time out`)
}

func AwaitGIDDownload(GIDS []string) error {
	client := NewAria2AgentClient(Conn)

	stream, err := client.AwaitDownload(context.Background(), &Param{
		GIDList: GIDS,
	})
	if err != nil {
		return err
	}

	defer CleanGID(GIDS)

	for {
		data, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch data.GetCode() {
		case 400:
			return errors.New(data.Msg)
		case 0:
			if data.GetFileInfo() != nil {
				if len(data.GetFileInfo()) == 0 {
					return errors.New(`Unexpected FileInfo`)
				}

				for _, v := range data.GetFileInfo() {
					if v.GetIsFinished() && v.GetGID() != `` {
						dao.DeletePendingByGID(v.GetGID())
					}
				}

				return nil
			} else {
				return errors.New(`Unexpected FileInfo`)
			}
		case 1:
		}
	}
	return errors.New(`time out`)
}

func CheckIsDownloadFinished(gids []string) (bool, error) {
	client := NewAria2AgentClient(Conn)

	defer CleanGID(gids)
	download, err := client.CheckDownload(context.Background(), &Param{
		GIDList: gids,
	})

	if err != nil {
		log.Logger.Error(err)
		return false, err
	}

	if download.GetCode() != 0 {
		return false, errors.New(`gid not exist`)
	}

	for _, v := range download.GetFileInfo() {
		if !v.GetIsFinished() {
			return false, nil
		}
	}

	return true, nil
}

func CleanGID(Gids []string) {
	if Gids == nil {
		return
	}

	if len(Gids) == 0 {
		return
	}

	err := dao.Clean(Gids)
	if err != nil {
		log.Logger.Error(err)
	}

}

func CleanGIDPointer(Gids *[]string) {
	if Gids == nil {
		return
	}

	if len(*Gids) == 0 {
		return
	}

	err := dao.Clean(*Gids)
	if err != nil {
		log.Logger.Error(err)
	}

}

func CleanGIDByParam(Param *Param) {
	if Param == nil {
		return
	}

	if Param.GetGIDList() == nil {
		return
	}

	if len(Param.GetGIDList()) == 0 {
		return
	}

	err := dao.Clean(Param.GetGIDList())
	if err != nil {
		log.Logger.Error(err)
	}
}
