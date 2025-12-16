package qbot

import (
	"fmt"
	"github.com/tencent-connect/botgo/interaction/webhook"
	"net/http"
	"os"
	"strings"
	"time"
)

func (s *Service) runHttp() error {
	conf := &s.config.Http
	http.HandleFunc(conf.Path, func(writer http.ResponseWriter, request *http.Request) {
		webhook.HTTPHandler(writer, request, &s.config.Credentials)
	})

	http.HandleFunc("/resource/", s.resourceService)

	errChan := make(chan error, 1)

	go func() {
		if conf.TLS.CertPath != "" && conf.TLS.KeyPath != "" {
			s.logger.Debug("Serving HTTPS on TLS")
			if err := http.ListenAndServeTLS(fmt.Sprintf("%s:%d", conf.Host, conf.Port), conf.TLS.CertPath, conf.TLS.KeyPath, nil); err != nil {
				errChan <- err
			}
		} else {
			s.logger.Debug("Serving HTTP")
			if err := http.ListenAndServe(fmt.Sprintf("%s:%d", conf.Host, conf.Port), nil); err != nil {
				errChan <- err
			}
		}

		close(errChan)
	}()

	select {
	case err := <-errChan:
		return err
	case <-time.After(500 * time.Millisecond):
	}
	return nil
}

func (s *Service) resourceService(writer http.ResponseWriter, request *http.Request) {
	// 检查请求路径是否符合 /resource/<id> 格式
	path := strings.TrimPrefix(request.URL.Path, "/resource")
	id := strings.Trim(path, "/#")
	s.logger.Debug("Getting resource %s through api", id)
	if id == "" || strings.Contains(id, "/") {
		http.Error(writer, "Invalid resource ID", http.StatusBadRequest)
		return
	}

	// 获取资源路径
	resourcePath, err := s.grb.LoadResourceFromID(id)
	if err != nil {
		http.Error(writer, "Resource not found", http.StatusNotFound)
		return
	}

	// 读取资源文件内容
	data, err := os.ReadFile(resourcePath)
	if err != nil {
		http.Error(writer, "Resource not found", http.StatusNotFound)
		return
	}

	// 返回资源文件内容
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(data)
}
