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
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type Resource struct {
	ID         string    // 资源唯一标识符
	FilePath   string    // 资源文件保存路径
	Downloaded time.Time // 资源下载时间
}

// SaveRemoteResource 保存资源文件，并更新资源索引
func (i *Instant) SaveRemoteResource(resourceURL string) (*Resource, error) {
	if resourceURL == "" {
		return nil, fmt.Errorf("resourceURL is empty")
	}

	// 生成资源文件保存路径
	currentTime := time.Now()
	resourceDirPath := path.Join("resources", currentTime.Format("2006/01.02"))

	// 下载资源文件并更新路径
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
		FilePath:   filePath,
		Downloaded: currentTime,
	}

	// 保存资源索引到数据库
	if err := i.saveResourceIndex(resource); err != nil {
		// 如果保存失败，回滚操作并删除下载的资源文件
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

	// Create directory structure if it doesn't exist
	if err := os.MkdirAll(resourceDirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create resource directory %s: %v", resourceDirPath, err)
	}

	resource := Resource{
		ID:         id,
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
	// 更新内存中的资源索引
	i.resourceMap[resource.ID] = resource

	// 如果数据库不存在，直接返回
	if !i.DatabaseExist() {
		return nil
	}

	// 将资源索引保存到数据库
	db := i.Database()
	_, err := db.Exec(fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS RESOURCES (
    ID TEXT PRIMARY KEY NOT NULL,
    PATH TEXT NOT NULL,
    TIME NUMERIC NOT NULL
);
INSERT INTO RESOURCES (ID, PATH, TIME)
VALUES ('%s', '%s', %d);`, resource.ID, resource.FilePath, resource.Downloaded.Unix()))

	if err != nil {
		// 如果保存失败，回滚操作
		delete(i.resourceMap, resource.ID)
		return err
	}

	return nil
}

func (i *Instant) ResourceExists(resourceID string) bool {
	_, ok := i.resourceMap[resourceID]
	if ok {
		return true
	}

	// Only check database if it exists
	if !i.DatabaseExist() {
		return false
	}

	_, err := i.GetResource(resourceID)
	if err == nil {
		return true
	}
	return false
}

// GetResource 根据资源ID获取资源信息
func (i *Instant) GetResource(resourceID string) (Resource, error) {
	// Check if database is available
	if !i.DatabaseExist() {
		return Resource{}, fmt.Errorf("database not available")
	}

	// 假设数据库连接对象为 i.Database()
	db := i.Database()

	// 查询数据库中的资源信息
	var (
		id       string
		filePath string
		timeUnix int64
	)
	err := db.QueryRow(`SELECT ID, PATH, TIME FROM RESOURCES WHERE ID = ?`, resourceID).
		Scan(&id, &filePath, &timeUnix)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// 如果没有找到资源，返回资源未找到错误
			return Resource{}, fmt.Errorf("resource with ID %s not found in database", resourceID)
		}
		// 其他数据库错误
		return Resource{}, fmt.Errorf("failed to query resource from database: %v", err)
	}

	return Resource{
		ID:         id,
		FilePath:   filePath,
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
	// 默认重试次数
	maxRetries := 5
	if len(retryCount) > 0 && retryCount[0] >= 0 {
		maxRetries = retryCount[0]
	}

	// 尝试下载文件，最多重试 maxRetries 次
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
	if err == nil {
		fileName = calcMd5(data) + "." + ext[0]
	}
	return
}

func calcMd5(data []byte) string {
	return fmt.Sprintf("%x", md5.Sum(data))
}
