package fileSystemService

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
	c, err := grpc.Dial(config.Conf.GetString(`worker.fs.grpc.addr`), grpc.WithInsecure())
	if err != nil {
		log.Logger.Error(err)
	}

	Conn = c
}

func FSListener(context.Context, *utils.TransmitterPipes) {

}

func CreateEntireFile(file []byte, path string) error {
	client := NewFileSystemClient(Conn)
	stream, err := client.FSCreate(context.Background())
	if err != nil {
		return err
	}

	if err := stream.Send(&Param{Source: &path, Data: &FileInfo{Index: 0, Data: file}}); err != nil {
		return err
	}

	result, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}

	if result == nil {
		return errors.New(`unexpected result`)
	}

	if result.GetCode() != 0 {
		return errors.New(result.GetMsg())
	}

	return nil
}

func DeleteFile(path string) error {
	client := NewFileSystemClient(Conn)

	result, err := client.FSDelete(context.Background(), &Param{
		Source: &path,
	})

	if err != nil {
		return err
	}

	if result.GetCode() != 0 {
		return errors.New(result.Msg)
	}

	return nil
}

func ListFile(path string) ([]File, error) {
	client := NewFileSystemClient(Conn)

	result, err := client.FSList(context.Background(), &Param{
		Source: &path,
	})

	if err != nil {
		return nil, err
	}

	if result.GetCode() != 0 {
		return nil, errors.New(result.Msg)
	}

	var Files []File

	for i := range result.GetFileInfo() {
		Files = append(Files, File{
			IsDir: result.GetFileInfo()[i].GetIsDir(),
			Name:  result.GetFileInfo()[i].GetName(),
		})
	}

	return Files, nil
}

type File struct {
	IsDir bool
	Name  string
}
