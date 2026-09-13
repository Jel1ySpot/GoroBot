package qbot

import (
	"context"
	"fmt"

	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
)

// UploadFileData 上传媒体文件数据并获取 FileInfo
func (s *Service) UploadFileData(id string, fileType uint64, data []byte) (*FileInfo, error) {
	info, ok := entity.ParseInfo(id)
	if !ok || info.Protocol != "qbot" || len(info.Args) < 2 {
		return nil, fmt.Errorf("invalid id info format")
	}
	switch info.Args[0] {
	case "group":
		return s.api.UploadFileData(context.Background(), "group", info.Args[1], fileType, data)
	case "user":
		return s.api.UploadFileData(context.Background(), "user", info.Args[1], fileType, data)
	default:
		return nil, fmt.Errorf("unsupported upload target %s", info.Args[0])
	}
}
