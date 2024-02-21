package sheet

import (
	"context"
	"fmt"

	"net/http"
	"strings"

	"example.com/debug"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type FileList struct {
	Name string
	Id   string
}
type DriveClient struct {
	Drv *drive.Service
}

func NewDriveClient(ctx context.Context, l *debug.DebugClient, client *http.Client) (*DriveClient, error) {
	var err error
	d := new(DriveClient)
	log = l
	d.Drv, err = drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Drive client: %v", err)
	}
	return d, err
}

func (d *DriveClient) ListDir(ctx context.Context, mimeType, sheetTitle string) ([]FileList, error) {
	log.V(1).Printf("mimeType:%v sheetTitle:%v\n", mimeType, sheetTitle)
	r, err := d.Drv.Files.List().Q("mimeType =\"" + mimeType + "\" and trashed = false").Fields("nextPageToken, files(id, name)").Do()
	if err != nil {
		return nil, err
	}
	if len(r.Files) == 0 {
		return nil, fmt.Errorf("no files found, mime type:%v", mimeType)
	}
	l := make([]FileList, 0)
	for _, prop := range r.Files {
		log.V(1).Printf("[%v:%v] (%v)\n", prop.Name, sheetTitle, prop.Id)
		if strings.Contains(prop.Name, sheetTitle) {
			f := FileList{
				Name: prop.Name,
				Id:   prop.Id,
			}
			l = append(l, f)
		}
	}

	log.V(1).Printf("fileList:%v\n", l)
	return l, nil

}
