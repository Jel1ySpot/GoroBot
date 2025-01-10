package GoroBot

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type Resource struct {
	ID         string    // 资源唯一标识符
	FilePath   string    // 资源文件保存路径
	Downloaded time.Time // 资源下载时间
}

// SaveResource 保存资源文件，并更新资源索引
func (i *Instant) SaveResource(resourceID string, resourceURL string) error {
	if resourceID == "" || resourceURL == "" {
		return fmt.Errorf("resourceID or resourceURL is empty")
	}

	if i.ResourceExists(resourceID) {
		return nil
	}

	// 生成资源文件保存路径
	currentTime := time.Now()
	resourceFilePath := path.Join("resources", currentTime.Format("2006/01.02"), resourceID)

	resource := Resource{
		ID:         resourceID,
		FilePath:   resourceFilePath,
		Downloaded: currentTime,
	}

	// 下载资源文件并更新路径
	downloadedFilePath, err := downloadFileWithRetry(resourceFilePath, resourceURL)
	if err != nil {
		return err
	}
	resource.FilePath = downloadedFilePath

	// 保存资源索引到数据库
	if err := i.saveResourceIndex(resource); err != nil {
		// 如果保存失败，回滚操作并删除下载的资源文件
		_ = os.Remove(downloadedFilePath)
		return err
	}

	return nil
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
	_, err := i.GetResource(resourceID)
	if err == nil {
		return true
	}
	return false
}

// GetResource 根据资源ID获取资源信息
func (i *Instant) GetResource(resourceID string) (Resource, error) {
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

// downloadFileWithRetry 尝试下载文件，带有重试机制
func downloadFileWithRetry(filePath string, url string, retryCount ...int) (string, error) {
	// 默认重试次数
	maxRetries := 5
	if len(retryCount) > 0 && retryCount[0] >= 0 {
		maxRetries = retryCount[0]
	}

	// 尝试下载文件，最多重试 maxRetries 次
	for retries := 0; retries < maxRetries; retries++ {
		downloadedFilePath, err := downloadFile(filePath, url)
		if err == nil {
			return downloadedFilePath, nil
		}
		if retries == maxRetries-1 {
			return "", fmt.Errorf("failed to download file after %d retries", maxRetries)
		}
	}

	return "", fmt.Errorf("unexpected error during file download")
}

// downloadFile 下载文件并保存到指定路径
func downloadFile(filePath string, url string) (string, error) {
	// 确保文件路径所在的目录存在
	if _, err := os.Stat(path.Dir(filePath)); os.IsNotExist(err) {
		if err := os.MkdirAll(path.Dir(filePath), os.ModePerm); err != nil {
			return "", err
		}
	}

	// 发送HTTP GET请求下载文件
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 获取文件扩展名
	ext, err := mime.ExtensionsByType(resp.Header.Get("Content-Type"))
	if err != nil {
		nameParse := strings.Split(resp.Header.Get("Content-Disposition"), ".")
		ext = []string{
			nameParse[len(nameParse)-1],
		}
	}
	filePath += ext[0]

	// 读取文件内容并保存
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return filePath, err
	}

	// 写入文件
	if err := os.WriteFile(filePath, data, os.ModePerm); err != nil {
		return filePath, err
	}

	return filePath, nil
}
