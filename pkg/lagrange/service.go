package lagrange

import (
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
	"github.com/Jel1ySpot/conic"
	"github.com/LagrangeDev/LagrangeGo/client"
	"github.com/google/uuid"
)

const (
	DefaultConfigPath = "conf/lagrange/"
)

type Service struct {
	ConfigPath string
	config     Config
	qqClient   *client.QQClient
	grb        *GoroBot.Instant
	owner      uint32
	status     botc.LoginStatus

	conic  *conic.Conic
	logger logger.Inst
}

func (s *Service) Name() string {
	return "Lagrange-adapter"
}

func Create() *Service {
	return &Service{
		conic:      conic.New(),
		status:     botc.Offline,
		ConfigPath: DefaultConfigPath,
	}
}

func (s *Service) getContext() *Context {
	return &Context{s}
}

func (s *Service) Init(grb *GoroBot.Instant) error {
	// https://blog.csdn.net/weixin_45760685/article/details/140629746
	_ = os.Setenv("GODEBUG", "tlsrsakex=1")

	s.grb = grb
	s.logger = grb.GetLogger()
	if id, ok := grb.GetOwner("qq"); ok {
		if uin, err := strconv.ParseUint(id, 10, 32); err == nil {
			s.owner = uint32(uin)
		}
	}

	if err := s.InitConic(); err != nil {
		return err
	}

	s.qqClient = client.NewClientEmpty()

	if err := s.login(); err != nil {
		return err
	}

	s.status = botc.Online

	grb.AddContext(s.getContext())

	return nil
}

func (s *Service) Release(grb *GoroBot.Instant) error {
	if s.qqClient != nil {
		if err := s.releaseQQClient(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) Protocol() string {
	return "lagrange"
}

func (s *Service) DownloadResourceFromRefLink(refLink string) (string, error) {
	values, err := urlpkg.ParseQuery(refLink)
	if err != nil {
		return "", fmt.Errorf("invalid ref link: %w", err)
	}

	rawURL := values.Get("url")
	if rawURL == "" {
		return "", fmt.Errorf("ref link missing url")
	}

	target := values.Get("target")
	if target == "" {
		ext := values.Get("ext")
		if ext == "" {
			if u, err := urlpkg.Parse(rawURL); err == nil {
				ext = path.Ext(u.Path)
			}
		}
		if ext == "" {
			ext = ".dat"
		} else if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		target = filepath.Join("resources", uuid.NewString()+ext)
	}

	resp, err := http.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("request resource failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read resource failed: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return "", fmt.Errorf("create cache dir failed: %w", err)
	}

	if err := os.WriteFile(target, data, 0644); err != nil {
		return "", fmt.Errorf("write resource failed: %w", err)
	}

	return target, nil
}

func (s *Service) releaseQQClient() error {
	s.qqClient.Release()

	s.saveSig()

	return nil
}

func (s *Service) saveSig() {
	data, err := s.qqClient.Sig().Marshal()
	if err != nil {
		s.logger.Error("marshal sig.bin err: %s", err)
		return
	}
	err = os.WriteFile(path.Join(s.ConfigPath, s.config.Account.SigPath), data, 0644)
	if err != nil {
		s.logger.Error("write sig.bin err: %s", err)
		return
	}
	s.logger.Success("sig saved into %s", path.Join(s.ConfigPath, s.config.Account.SigPath))
}
