package qbot

import (
	"fmt"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
)

func (s *Service) loadImageData(data []byte, ext string) string {
	if id, err := s.grb.SaveResourceData(data, ext); err != nil {
		return ""
	} else {
		return id
	}
}

var (
	ImageFileType uint64 = 1
	VideoFileType        = 2
	VoiceFileType        = 3
)

type (
	FileUpload struct {
		FileType uint64 `json:"file_type,omitempty"` // 业务类型，图片，文件，语音，视频 文件类型，取值:1图片,2视频,3语音(目前语音只支持silk格式)
		FileData []byte `json:"file_data,omitempty"`
	}

	FileInfo struct {
		ID       string `json:"id,omitempty"`
		FileUUID string `json:"file_uuid,omitempty"`
		FileInfo []byte `json:"file_info,omitempty"`
		TTL      uint   `json:"ttl,omitempty"`
	}
)

func (s *Service) UploadFileData(id string, fileType uint64, data []byte) (*FileInfo, error) {
	info, ok := entity.ParseInfo(id)
	if !ok || info.Protocol != "qbot" {
		return nil, fmt.Errorf("invalid id info format")
	}
	endPoint := ""
	switch info.Args[0] {
	case "group":
		endPoint = "/v2/groups/{id}/files"
	case "user":
		endPoint = "/v2/users/{id}/files"
	default:
		return nil, fmt.Errorf("unsupport upload method")
	}

	body := FileUpload{
		FileType: fileType,
		FileData: data,
	}

	resp, err := NativePost(s.api, endPoint, body, &FileInfo{}, map[string]string{"id": info.Args[1]})
	if err != nil {
		return nil, err
	}
	return resp.(*FileInfo), nil
}

func (s *Service) UploadImageData(id string, data []byte) (*FileInfo, error) {
	return s.UploadFileData(id, ImageFileType, data)
}
