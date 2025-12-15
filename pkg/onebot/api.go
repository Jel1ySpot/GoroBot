package onebot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// OneBot API response structures
type APIResponse struct {
	Status  string      `json:"status"`
	RetCode int         `json:"retcode"`
	Data    interface{} `json:"data"`
	Echo    interface{} `json:"echo"`
}

type LoginInfo struct {
	UserID   int64  `json:"user_id"`
	Nickname string `json:"nickname"`
}

type SendMessageResponse struct {
	MessageID int64 `json:"message_id"`
}

type Friend struct {
	UserID   int64  `json:"user_id"`
	Nickname string `json:"nickname"`
	Remark   string `json:"remark"`
}

type Group struct {
	GroupID        int64  `json:"group_id"`
	GroupName      string `json:"group_name"`
	MemberCount    int32  `json:"member_count"`
	MaxMemberCount int32  `json:"max_member_count"`
}

type GroupMember struct {
	GroupID         int64  `json:"group_id"`
	UserID          int64  `json:"user_id"`
	Nickname        string `json:"nickname"`
	Card            string `json:"card"`
	Sex             string `json:"sex"`
	Age             int32  `json:"age"`
	Area            string `json:"area"`
	JoinTime        int32  `json:"join_time"`
	LastSentTime    int32  `json:"last_sent_time"`
	Level           string `json:"level"`
	Role            string `json:"role"`
	Unfriendly      bool   `json:"unfriendly"`
	Title           string `json:"title"`
	TitleExpireTime int32  `json:"title_expire_time"`
	CardChangeable  bool   `json:"card_changeable"`
}

type Status struct {
	Online bool `json:"online"`
	Good   bool `json:"good"`
}

// API client methods

func (s *Service) makeAPIRequest(action string, params map[string]interface{}) (*APIResponse, error) {
	switch s.config.Mode {
	case "http":
		return s.makeHTTPRequest(action, params)
	case "ws", "ws_reverse":
		return s.makeWebSocketRequest(action, params)
	default:
		return nil, fmt.Errorf("unsupported connection mode for API calls: %s", s.config.Mode)
	}
}

func (s *Service) makeHTTPRequest(action string, params map[string]interface{}) (*APIResponse, error) {
	baseURL := s.config.HTTP.PostURL

	var resp *http.Response
	var err error

	if len(params) == 0 {
		// GET request
		reqURL := fmt.Sprintf("%s/%s", baseURL, action)
		if s.config.HTTP.AccessToken != "" {
			reqURL += "?access_token=" + url.QueryEscape(s.config.HTTP.AccessToken)
		}

		s.logger.Debug("Making HTTP GET request to %s", reqURL)
		resp, err = s.httpClient.Get(reqURL)
	} else {
		// POST request with JSON body
		reqURL := fmt.Sprintf("%s/%s", baseURL, action)

		jsonData, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %v", err)
		}

		req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		if s.config.HTTP.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer "+s.config.HTTP.AccessToken)
		}

		s.logger.Debug("Making HTTP POST request to %s with data: %s", reqURL, string(jsonData))
		resp, err = s.httpClient.Do(req)
	}

	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	s.logger.Debug("HTTP response status: %d, body: %s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case 401:
			return nil, fmt.Errorf("authentication failed - check access token")
		case 403:
			return nil, fmt.Errorf("access forbidden - invalid access token")
		case 404:
			return nil, fmt.Errorf("API endpoint not found - check OneBot server configuration")
		case 406:
			return nil, fmt.Errorf("unsupported content type")
		default:
			return nil, fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
		}
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %v", err)
	}

	if apiResp.Status == "failed" {
		return nil, fmt.Errorf("OneBot API call failed with retcode %d", apiResp.RetCode)
	}

	return &apiResp, nil
}

func (s *Service) makeWebSocketRequest(action string, params map[string]interface{}) (*APIResponse, error) {
	s.apiConnMu.Lock()         // 加锁
	defer s.apiConnMu.Unlock() // 解锁

	if s.apiConn == nil {
		return nil, fmt.Errorf("WebSocket API connection not available")
	}

	// Create WebSocket API request
	request := map[string]interface{}{
		"action": action,
		"params": params,
		"echo":   time.Now().UnixNano(), // Use timestamp as echo
	}

	if err := s.apiConn.WriteJSON(request); err != nil {
		return nil, fmt.Errorf("failed to send WebSocket request: %v", err)
	}

	// Read response
	var response APIResponse
	if err := s.apiConn.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to read WebSocket response: %v", err)
	}

	if response.Status == "failed" {
		return nil, fmt.Errorf("API call failed with retcode %d", response.RetCode)
	}

	return &response, nil
}

// Specific API methods

func (s *Service) getLoginInfo() (*LoginInfo, error) {
	resp, err := s.makeAPIRequest("get_login_info", nil)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal login info: %v", err)
	}

	var loginInfo LoginInfo
	if err := json.Unmarshal(data, &loginInfo); err != nil {
		return nil, fmt.Errorf("failed to parse login info: %v", err)
	}

	return &loginInfo, nil
}

func (s *Service) sendPrivateMessage(userID int64, message interface{}) (*SendMessageResponse, error) {
	params := map[string]interface{}{
		"user_id": userID,
		"message": message,
	}

	resp, err := s.makeAPIRequest("send_private_msg", params)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal send message response: %v", err)
	}

	var sendResp SendMessageResponse
	if err := json.Unmarshal(data, &sendResp); err != nil {
		return nil, fmt.Errorf("failed to parse send message response: %v", err)
	}

	return &sendResp, nil
}

func (s *Service) sendGroupMessage(groupID int64, message interface{}) (*SendMessageResponse, error) {
	params := map[string]interface{}{
		"group_id": groupID,
		"message":  message,
	}

	resp, err := s.makeAPIRequest("send_group_msg", params)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal send message response: %v", err)
	}

	var sendResp SendMessageResponse
	if err := json.Unmarshal(data, &sendResp); err != nil {
		return nil, fmt.Errorf("failed to parse send message response: %v", err)
	}

	return &sendResp, nil
}

func (s *Service) getFriendList() ([]Friend, error) {
	// Try to return from cache first
	s.cache.friendListMu.RLock()
	if len(s.cache.friendList) > 0 && time.Since(s.cache.lastFriendUpdate) < CacheUpdateInterval {
		// Convert map to slice
		friends := make([]Friend, 0, len(s.cache.friendList))
		for _, friend := range s.cache.friendList {
			friends = append(friends, friend)
		}
		s.cache.friendListMu.RUnlock()
		s.logger.Debug("Returning cached friend list (%d friends)", len(friends))
		return friends, nil
	}
	s.cache.friendListMu.RUnlock()

	// Cache is empty or expired, fetch fresh data
	s.logger.Debug("Fetching fresh friend list from OneBot API...")
	friends, err := s.fetchFriendList()
	if err != nil {
		return nil, err
	}

	// Update cache
	friendMap := make(map[int64]Friend)
	for _, friend := range friends {
		friendMap[friend.UserID] = friend
	}

	s.cache.friendListMu.Lock()
	s.cache.friendList = friendMap
	s.cache.lastFriendUpdate = time.Now()
	s.cache.friendListMu.Unlock()

	s.logger.Debug("Friend list fetched and cached (%d friends)", len(friends))
	return friends, nil
}

func (s *Service) fetchFriendList() ([]Friend, error) {
	resp, err := s.makeAPIRequest("get_friend_list", nil)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal friend list: %v", err)
	}

	var friends []Friend
	if err := json.Unmarshal(data, &friends); err != nil {
		return nil, fmt.Errorf("failed to parse friend list: %v", err)
	}

	return friends, nil
}

func (s *Service) getGroupList() ([]Group, error) {
	// Try to return from cache first
	s.cache.groupListMu.RLock()
	if len(s.cache.groupList) > 0 && time.Since(s.cache.lastGroupUpdate) < CacheUpdateInterval {
		// Convert map to slice
		groups := make([]Group, 0, len(s.cache.groupList))
		for _, group := range s.cache.groupList {
			groups = append(groups, group)
		}
		s.cache.groupListMu.RUnlock()
		s.logger.Debug("Returning cached group list (%d groups)", len(groups))
		return groups, nil
	}
	s.cache.groupListMu.RUnlock()

	// Cache is empty or expired, fetch fresh data
	s.logger.Debug("Fetching fresh group list from OneBot API...")
	groups, err := s.fetchGroupList()
	if err != nil {
		return nil, err
	}

	// Update cache
	groupMap := make(map[int64]Group)
	for _, group := range groups {
		groupMap[group.GroupID] = group
	}

	s.cache.groupListMu.Lock()
	s.cache.groupList = groupMap
	s.cache.lastGroupUpdate = time.Now()
	s.cache.groupListMu.Unlock()

	s.logger.Debug("Group list fetched and cached (%d groups)", len(groups))
	return groups, nil
}

func (s *Service) fetchGroupList() ([]Group, error) {
	resp, err := s.makeAPIRequest("get_group_list", nil)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal group list: %v", err)
	}

	var groups []Group
	if err := json.Unmarshal(data, &groups); err != nil {
		return nil, fmt.Errorf("failed to parse group list: %v", err)
	}

	return groups, nil
}

func (s *Service) getGroupMemberInfo(groupID, userID int64, noCache bool) (*GroupMember, error) {
	params := map[string]interface{}{
		"group_id": groupID,
		"user_id":  userID,
		"no_cache": noCache,
	}

	resp, err := s.makeAPIRequest("get_group_member_info", params)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal group member info: %v", err)
	}

	var member GroupMember
	if err := json.Unmarshal(data, &member); err != nil {
		return nil, fmt.Errorf("failed to parse group member info: %v", err)
	}

	return &member, nil
}

func (s *Service) getStatus() (*Status, error) {
	resp, err := s.makeAPIRequest("get_status", nil)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status: %v", err)
	}

	var status Status
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("failed to parse status: %v", err)
	}

	return &status, nil
}

// getCachedGroupInfo retrieves group info from cache by ID
func (s *Service) getCachedGroupInfo(groupID int64) (Group, bool) {
	s.cache.groupListMu.RLock()
	defer s.cache.groupListMu.RUnlock()

	group, exists := s.cache.groupList[groupID]
	return group, exists
}

// getCachedFriendInfo retrieves friend info from cache by ID
func (s *Service) getCachedFriendInfo(userID int64) (Friend, bool) {
	s.cache.friendListMu.RLock()
	defer s.cache.friendListMu.RUnlock()

	friend, exists := s.cache.friendList[userID]
	return friend, exists
}
