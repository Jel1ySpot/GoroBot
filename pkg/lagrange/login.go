package lagrange

import (
	"fmt"
	"github.com/LagrangeDev/LagrangeGo/client"
	"github.com/LagrangeDev/LagrangeGo/client/auth"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

func (s *Service) login() error {
	appInfoConfig := strings.Split(s.config.AppInfo, " ")
	appInfo := auth.AppList[appInfoConfig[0]][appInfoConfig[1]]

	qqClient := s.qqClient
	qqClient.SetLogger(s.logger)
	qqClient.UseVersion(appInfo)
	qqClient.AddSignServer(s.config.SignServerUrl)

	devicePath := path.Join(DefaultConfigPath, "device.json")
	s.logger.Debug("Lagrange 读取设备配置文件: %s", devicePath)
	deviceInfo, err := auth.LoadOrSaveDevice(devicePath)
	if err != nil {
		return err
	}
	qqClient.UseDevice(deviceInfo)

	sigPath := path.Join(DefaultConfigPath, s.config.Account.SigPath)
	s.logger.Debug("Lagrange 读取签名凭证文件: %s", sigPath)
	data, err := os.ReadFile(sigPath)
	if err == nil {
		s.logger.Debug("Lagrange 签名凭证文件读取成功 (%d 字节)", len(data))
		sig, err := auth.UnmarshalSigInfo(data, true)
		if err != nil {
			s.logger.Warning("load sig error: %s", err)
		} else {
			qqClient.UseSig(sig)
		}
	}

	if err := s.eventSubscribe(); err != nil {
		return err
	}

	err = func(c *client.QQClient) error {
		s.logger.Error("try FastLogin")
		if err := c.FastLogin(); err != nil {
			s.logger.Failed("fastLogin fail: %s", err)
		} else {
			return nil
		}

		s.logger.Error("login with qrcode")
		c = client.NewClient(0, "")
		s.qqClient = c
		_, uri, err := c.FetchQRCodeDefault()
		if err != nil {
			return err
		}
		s.logger.Error("QRCode: https://api.qrserver.com/v1/create-qr-code/?data=%s", url.QueryEscape(uri))
		for {
			retCode, err := c.GetQRCodeResult()
			if err != nil {
				return err
			}
			if retCode.Waitable() {
				time.Sleep(3 * time.Second)
				continue
			}
			if !retCode.Success() {
				return fmt.Errorf("%s", retCode.Name())
			}
			break
		}
		_, err = c.QRCodeLogin()
		return err
	}(qqClient)

	if err != nil {
		return err
	}
	s.logger.Success("login successed")

	s.saveSig()

	return nil
}
