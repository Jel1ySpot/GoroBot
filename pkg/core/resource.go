package GoroBot

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	urlpkg "net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Resource struct {
	ID         string    // 资源唯一标识符
	Protocol   string    // 资源所属协议
	RefLink    string    // 资源引用信息
	FilePath   string    // 资源文件保存路径
	Error      string    // 下载错误信息
	Downloaded time.Time // 资源下载时间
}

// ResourceDownloader 由各协议适配器实现，用于根据 refLink 下载资源到本地
// SaveResourceLink 存储资源引用并返回生成的资源 ID
func (i *Instant) SaveResourceLink(protocol string, refLink string) string {
	id := uuid.NewString()
	now := time.Now()
	res := Resource{
		ID:         id,
		Protocol:   protocol,
		RefLink:    refLink,
		Downloaded: now,
	}
	i.resourceMap[id] = res

	if !i.DatabaseExist() {
		return id
	}

	db := i.Database()
	if err := ensureResourceTable(db); err != nil {
		i.logger.Error("ensure resource table failed: %v", err)
		return id
	}

	if _, err := db.Exec(`INSERT INTO RESOURCES (ID, PROTOCOL, REF_LINK, PATH, ERROR, TIME) VALUES (?, ?, ?, ?, ?, ?)`, res.ID, res.Protocol, res.RefLink, "", "", res.Downloaded.Unix()); err != nil {
		i.logger.Error("insert resource link failed: %v", err)
	}

	return id
}

// LoadResourceFromID 使用资源 ID 加载本地文件路径，必要时通过协议适配器下载
func (i *Instant) LoadResourceFromID(id string) (string, error) {
	res, ok := i.resourceMap[id]

	if dbRes, err := i.loadResourceFromDB(id); err == nil {
		res = dbRes
		ok = true
	}

	if !ok {
		return "", fmt.Errorf("resource id %s not found", id)
	}

	if res.FilePath != "" {
		if _, err := os.Stat(res.FilePath); err == nil {
			return res.FilePath, nil
		}
	}

	if res.Error != "" {
		return "", fmt.Errorf(res.Error)
	}

	downloader, ok := i.contexts[res.Protocol]
	if !ok {
		return "", fmt.Errorf("no downloader registered for protocol %s", res.Protocol)
	}

	targetPath := buildTargetPath(id, res.RefLink)
	refLink := withTarget(res.RefLink, targetPath)

	path, err := downloader.DownloadResourceFromRefLink(refLink)
	if err != nil {
		_ = i.updateResourcePathOrError(id, targetPath, err.Error())
		return "", err
	}

	if path == "" {
		path = targetPath
	}

	if err := i.updateResourcePathOrError(id, path, ""); err != nil {
		return "", err
	}

	return path, nil
}

func (i *Instant) loadResourceFromDB(id string) (Resource, error) {
	if !i.DatabaseExist() {
		return Resource{}, fmt.Errorf("database not available")
	}
	db := i.Database()
	if err := ensureResourceTable(db); err != nil {
		return Resource{}, err
	}

	var (
		protocol string
		refLink  string
		path     string
		errMsg   string
		timeUnix int64
	)
	err := db.QueryRow(`SELECT PROTOCOL, REF_LINK, PATH, ERROR, TIME FROM RESOURCES WHERE ID = ?`, id).
		Scan(&protocol, &refLink, &path, &errMsg, &timeUnix)
	if err != nil {
		return Resource{}, err
	}

	return Resource{
		ID:         id,
		Protocol:   protocol,
		RefLink:    refLink,
		FilePath:   path,
		Error:      errMsg,
		Downloaded: time.Unix(timeUnix, 0),
	}, nil
}

func (i *Instant) updateResourcePathOrError(id string, path string, errMsg string) error {
	res, ok := i.resourceMap[id]
	if ok {
		res.FilePath = path
		res.Error = errMsg
		res.Downloaded = time.Now()
		i.resourceMap[id] = res
	}

	if !i.DatabaseExist() {
		return nil
	}
	db := i.Database()
	if err := ensureResourceTable(db); err != nil {
		return err
	}

	_, err := db.Exec(`UPDATE RESOURCES SET PATH = ?, ERROR = ?, TIME = ? WHERE ID = ?`, path, errMsg, time.Now().Unix(), id)
	return err
}

// SaveRemoteResource 保存资源文件，并更新资源索引
func (i *Instant) SaveRemoteResource(resourceURL string) (*Resource, error) {
	if resourceURL == "" {
		return nil, fmt.Errorf("resourceURL is empty")
	}

	currentTime := time.Now()
	resourceDirPath := path.Join("resources", currentTime.Format("2006/01.02"))

	data, fileName, err := downloadFileWithRetry(resourceURL)
	if err != nil {
		return nil, err
	}

	id := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	if i.ResourceExists(id) {
		return nil, fmt.Errorf("resource %s already exists", resourceURL)
	}

	filePath := path.Join(resourceDirPath, fileName)

	if _, err := os.Stat(resourceDirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(resourceDirPath, os.ModePerm); err != nil {
			return nil, fmt.Errorf("error creating resource dir %s: %v", resourceDirPath, err)
		}
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write resource file %s: %v", filePath, err)
	}

	resource := Resource{
		ID:         id,
		Protocol:   "local",
		RefLink:    "",
		FilePath:   filePath,
		Downloaded: currentTime,
	}

	if err := i.saveResourceIndex(resource); err != nil {
		_ = os.Remove(filePath)
		return nil, err
	}

	return &resource, nil
}

func (i *Instant) SaveResourceData(data []byte, ext string) (string, error) {
	hash := md5.Sum(data)
	id := hex.EncodeToString(hash[:])

	if i.ResourceExists(id) {
		return id, nil
	}
	currentTime := time.Now()
	resourceDirPath := path.Join("resources", currentTime.Format("2006/01.02"))
	resourceFilePath := path.Join(resourceDirPath, id+"."+ext)

	if err := os.MkdirAll(resourceDirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create resource directory %s: %v", resourceDirPath, err)
	}

	resource := Resource{
		ID:         id,
		Protocol:   "local",
		FilePath:   resourceFilePath,
		Downloaded: currentTime,
	}

	if err := os.WriteFile(resourceFilePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write resource file %s: %v", resourceFilePath, err)
	}

	if err := i.saveResourceIndex(resource); err != nil {
		return "", err
	}

	return id, nil
}

// saveResourceIndex 保存资源的元数据索引到内存和数据库
func (i *Instant) saveResourceIndex(resource Resource) error {
	i.resourceMap[resource.ID] = resource

	if !i.DatabaseExist() {
		return nil
	}

	db := i.Database()
	if err := ensureResourceTable(db); err != nil {
		delete(i.resourceMap, resource.ID)
		return err
	}

	_, err := db.Exec(`
INSERT INTO RESOURCES (ID, PROTOCOL, REF_LINK, PATH, ERROR, TIME)
VALUES (?, ?, ?, ?, ?, ?);`, resource.ID, resource.Protocol, resource.RefLink, resource.FilePath, resource.Error, resource.Downloaded.Unix())
	if err != nil {
		delete(i.resourceMap, resource.ID)
		return err
	}

	return nil
}

func (i *Instant) ResourceExists(resourceID string) bool {
	if _, ok := i.resourceMap[resourceID]; ok {
		return true
	}

	if !i.DatabaseExist() {
		return false
	}

	if _, err := i.GetResource(resourceID); err == nil {
		return true
	}
	return false
}

// GetResource 根据资源ID获取资源信息
func (i *Instant) GetResource(resourceID string) (Resource, error) {
	if !i.DatabaseExist() {
		return Resource{}, fmt.Errorf("database not available")
	}

	db := i.Database()
	if err := ensureResourceTable(db); err != nil {
		return Resource{}, err
	}

	var (
		id       string
		protocol string
		refLink  string
		filePath string
		errMsg   string
		timeUnix int64
	)
	err := db.QueryRow(`SELECT ID, PROTOCOL, REF_LINK, PATH, ERROR, TIME FROM RESOURCES WHERE ID = ?`, resourceID).
		Scan(&id, &protocol, &refLink, &filePath, &errMsg, &timeUnix)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Resource{}, fmt.Errorf("resource with ID %s not found in database", resourceID)
		}
		return Resource{}, fmt.Errorf("failed to query resource from database: %v", err)
	}

	return Resource{
		ID:         id,
		Protocol:   protocol,
		RefLink:    refLink,
		FilePath:   filePath,
		Error:      errMsg,
		Downloaded: time.Unix(timeUnix, 0),
	}, nil
}

func (i *Instant) GetResourceData(resourceID string) ([]byte, error) {
	resource, err := i.GetResource(resourceID)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(resource.FilePath)
}

// downloadFileWithRetry 尝试下载文件，带有重试机制
func downloadFileWithRetry(url string, retryCount ...int) (data []byte, fileName string, err error) {
	maxRetries := 5
	if len(retryCount) > 0 && retryCount[0] >= 0 {
		maxRetries = retryCount[0]
	}

	for retries := 0; retries < maxRetries; retries++ {
		data, fileName, err = downloadFile(url)
		if err == nil {
			return
		}
	}

	return
}

func downloadFile(url string) (data []byte, fileName string, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	ext, err := mime.ExtensionsByType(resp.Header.Get("Content-Type"))
	if name := resp.Header.Get("Content-Disposition"); name != "" {
		Ext := strings.TrimLeft(path.Ext(name), ".")
		ext = []string{
			Ext,
		}
	}

	data, err = io.ReadAll(resp.Body)
	if err == nil && len(ext) > 0 {
		fileName = calcMd5(data) + "." + ext[0]
	}
	return
}

func calcMd5(data []byte) string {
	return fmt.Sprintf("%x", md5.Sum(data))
}

func ensureResourceTable(db *sql.DB) error {
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS RESOURCES (
    ID TEXT PRIMARY KEY NOT NULL,
    PROTOCOL TEXT,
    REF_LINK TEXT,
    PATH TEXT,
    ERROR TEXT,
    TIME NUMERIC NOT NULL
);`); err != nil {
		return err
	}

	columns := map[string]bool{}
	rows, err := db.Query(`PRAGMA table_info(RESOURCES);`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		columns[strings.ToUpper(name)] = true
	}

	for _, col := range []string{"PROTOCOL", "REF_LINK", "PATH", "ERROR", "TIME"} {
		if !columns[col] {
			if _, err := db.Exec(fmt.Sprintf(`ALTER TABLE RESOURCES ADD COLUMN %s TEXT;`, col)); err != nil {
				return err
			}
		}
	}

	return nil
}

func buildTargetPath(id string, refLink string) string {
	ext := ".dat"
	if values, err := urlpkg.ParseQuery(refLink); err == nil {
		if e := values.Get("ext"); e != "" {
			if strings.HasPrefix(e, ".") {
				ext = e
			} else {
				ext = "." + e
			}
		} else if rawURL := values.Get("url"); rawURL != "" {
			if u, err := urlpkg.Parse(rawURL); err == nil {
				if path.Ext(u.Path) != "" {
					ext = path.Ext(u.Path)
				}
			}
		}
	}

	return filepath.Join("resources", fmt.Sprintf("%s%s", id, ext))
}

func withTarget(refLink string, target string) string {
	values, err := urlpkg.ParseQuery(refLink)
	if err != nil {
		return refLink
	}
	values.Set("target", target)
	return values.Encode()
}
