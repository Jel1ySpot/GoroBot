# 资源文件管理
位于 `pkg/core/resource.go`，是 GoroBot 示例下的方法，为统一管理资源文件提供了接口。
文件存放于工作目录的 `resources/` 下。

## Resource
- ID `string` 资源唯一标识符
- FilePath `string` 资源文件保存路径
- Downloaded `time.Time` 资源下载时间

### grb.SaveResource(resourceID string, resourceURL string) error
保存资源文件。

### grb.ResourceExists(resourceID string) bool
如果 ID 所指的资源存在，返回 `true`，否则返回 `false` 。

### grb.GetResource(resourceID string) ([]byte, error)
获取 ID 所指向的资源信息
