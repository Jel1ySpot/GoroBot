package lagrange

import (
	"fmt"
	LgrMessage "github.com/LagrangeDev/LagrangeGo/message"
)

func (s *Service) parseResources(elements []LgrMessage.IMessageElement, groupUin uint32) {
	for _, elem := range elements {
		switch elem := elem.(type) {
		case *LgrMessage.VoiceElement:
			if err := s.saveVoiceResource(elem, groupUin); err != nil {
				s.logger.Warning("save voice err: %v", err)
			}
		case *LgrMessage.ImageElement:
			if err := s.saveImageResource(elem); err != nil {
				s.logger.Warning("save image err: %v", err)
			}
		case *LgrMessage.ShortVideoElement:
			if err := s.saveVideoResource(elem, groupUin > 0); err != nil {
				s.logger.Warning("save short video err: %v", err)
			}
		}
	}
}

func (s *Service) saveImageResource(elem *LgrMessage.ImageElement) error {
	var (
		imageElem *LgrMessage.ImageElement
		err       error
	)

	if elem.IsGroup {
		imageElem, err = s.qqClient.QueryGroupImage(elem.Md5, elem.FileUUID)
	} else {
		imageElem, err = s.qqClient.QueryFriendImage(elem.Md5, elem.FileUUID)
	}

	if err != nil {
		return err
	}

	return s.bot.SaveResource(fmt.Sprintf("%x", elem.Md5), imageElem.URL)
}

func (s *Service) saveVoiceResource(elem *LgrMessage.VoiceElement, groupUin uint32) error {
	var (
		url string
		err error
	)
	if groupUin > 0 {
		url, err = s.qqClient.GetGroupRecordURL(groupUin, elem.Node)
	} else {
		url, err = s.qqClient.GetPrivateRecordURL(elem.Node)
	}
	if err != nil {
		return err
	}

	return s.bot.SaveResource(fmt.Sprintf("%x", elem.Md5), url)
}

func (s *Service) saveVideoResource(elem *LgrMessage.ShortVideoElement, isGroup bool) error {
	url, err := s.qqClient.GetVideoURL(isGroup, elem.UUID)
	if err != nil {
		return err
	}
	return s.bot.SaveResource(fmt.Sprintf("%x", elem.Md5), url)
}
