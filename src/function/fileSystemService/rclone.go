package fileSystemService

import (
	"avalon-core/src/log"
	"avalon-core/src/utils"
	"context"
	"errors"
	jsoniter "github.com/json-iterator/go"
	"sort"
	"strings"
	"sync"
)

func RcloneMkdir(path string) error {
	client := NewFileSystemClient(Conn)

	stream, err := client.RCloneMkdir(context.Background(), &Param{
		Source: &path,
	})

	if err != nil {
		return err
	}

	for {
		recv, err := stream.Recv()
		if err != nil {
			return err
		}

		switch recv.GetCode() {
		case 400:
			return errors.New(recv.GetMsg())
		case 0:
			return nil
		case 1:
			continue
		}
	}
}

func RcloneLink(path string) (string, error) {
	client := NewFileSystemClient(Conn)
	stream, err := client.RCloneLink(context.Background(), &Param{
		Source: &path,
	})

	if err != nil {
		return ``, err
	}

	for {
		recv, err := stream.Recv()
		if err != nil {
			return ``, err
		}

		switch recv.GetCode() {
		case 400:
			return ``, errors.New(recv.GetMsg())
		case 0:
			link := strings.TrimSuffix(utils.ByteToString(recv.GetData()), "\n")
			if link == `` {
				return ``, errors.New(`unexpected output`)
			}

			return link, err
		case 1:
			continue
		}
	}
}

// RcloneMove Dir only
func RcloneMove(local, remoteDirPath string) error {
	client := NewFileSystemClient(Conn)

	stream, err := client.RCloneMove(context.Background(), &Param{
		Source:      &local,
		Destination: &remoteDirPath,
		Args: []RcloneArguments{
			RcloneArguments_NO_TRAVERSE,
			RcloneArguments_DELETE_EMPTY_SRC_DIRS,
			RcloneArguments_CREATE_EMPTY_SRC_DIRS,
		},
	})

	if err != nil {
		return err
	}

	for {
		recv, err := stream.Recv()
		if err != nil {
			return err
		}

		switch recv.GetCode() {
		case 400:
			return errors.New(recv.GetMsg())
		case 0:
			return nil
		case 1:
			continue
		}
	}
}

func RcloneCopy(local, remoteDirPath string) error {
	client := NewFileSystemClient(Conn)

	stream, err := client.RCloneCopy(context.Background(), &Param{
		Source:      &local,
		Destination: &remoteDirPath,
		Args: []RcloneArguments{
			RcloneArguments_NO_TRAVERSE,
			RcloneArguments_CREATE_EMPTY_SRC_DIRS,
		},
	})

	if err != nil {
		return err
	}

	for {
		recv, err := stream.Recv()
		if err != nil {
			return err
		}

		switch recv.GetCode() {
		case 400:
			return errors.New(recv.GetMsg())
		case 0:
			return nil
		case 1:
			continue
		}
	}
}

func RcloneGetDrive(kind string) (string, error) {
	client := NewFileSystemClient(Conn)

	drives := []string{}
	available := []string{}

	remotes, err := client.RCloneListRemotes(context.Background(), &Param{Source: &kind})
	if err != nil {
		return ``, err
	}

	for {
		recv, err := remotes.Recv()
		if err != nil {
			return ``, err
		}

		switch recv.GetCode() {
		case 400:
			return ``, errors.New(recv.GetMsg())
		case 0:
			drive := strings.Split(utils.ByteToString(recv.GetData()), "\n")
			for i := range drive {
				if strings.Contains(drive[i], kind) {
					drives = append(drives, drive[i])
				}
			}
			break
		case 1:
			continue
		}
		break
	}
	var WG sync.WaitGroup
	for i := range drives {
		WG.Add(1)
		go func(index int) {
			c := NewFileSystemClient(Conn)
			about, err := c.RCloneAbout(context.Background(), &Param{
				Source: &(drives[index]),
				Args:   []RcloneArguments{RcloneArguments_ABOUT_FORMAT_JSON},
			})

			defer WG.Done()
			if err != nil {
				log.Logger.Error(err)
				return
			}

			for {
				recv, err := about.Recv()
				if err != nil {
					log.Logger.Error(err)
					return
				}

				switch recv.GetCode() {
				case 400:
					log.Logger.Error(recv.GetMsg())
					return
				case 0:
					var info DriveStatus

					err := jsoniter.Unmarshal(recv.GetData(), &info)
					if err != nil {
						log.Logger.Error(err)
						return
					}

					if info.Free > (30 * 1073741824) {
						available = append(available, drives[i])
					}
					return
				case 1:
					continue
				}
			}
		}(i)

		WG.Wait()
	}

	sort.Strings(available)

	if len(available) == 0 {
		return ``, errors.New(`available drive not found`)
	}

	return available[0], nil
}

type DriveStatus struct {
	Total   int64 `json:"total"`
	Used    int64 `json:"used"`
	Trashed int64 `json:"trashed"`
	Free    int64 `json:"free"`
}
