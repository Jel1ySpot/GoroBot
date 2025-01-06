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

	deviceInfo, err := auth.LoadOrSaveDevice(path.Join(DefaultConfigPath, "device.json"))
	if err != nil {
		return err
	}
	qqClient.UseDevice(deviceInfo)

	data, err := os.ReadFile(path.Join(DefaultConfigPath, s.config.Account.SigPath))
	if err == nil {
		sig, err := auth.UnmarshalSigInfo(data, true)
		if err != nil {
			s.logger.Warning("load sig error: %s", err)
		} else {
			qqClient.UseSig(sig)
		}
	}

	// TODO: 消息订阅
	if err := s.eventSubscribe(); err != nil {
		return err
	}

	err = func(c *client.QQClient) error {
		s.logger.Info("try FastLogin")
		if err := c.FastLogin(); err != nil {
			s.logger.Warning("fastLogin fail: %s", err)
		} else {
			return nil
		}

		s.logger.Info("login with qrcode")
		_, uri, err := c.FetchQRCodeDefault()
		if err != nil {
			return err
		}
		s.logger.Warning("QRCode: https://api.qrserver.com/v1/create-qr-code/?data=%s", url.QueryEscape(uri))
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
				return fmt.Errorf(retCode.Name())
			}
			break
		}
		_, err = c.QRCodeLogin()
		return err
	}(qqClient)

	if err != nil {
		return err
	}
	s.logger.Info("login successed")

	return nil
}
