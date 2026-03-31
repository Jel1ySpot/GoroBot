# 资源文件管理
位于 `pkg/core/resource.go`，是 GoroBot 实例下的方法，为统一管理资源文件提供了接口。
文件存放于工作目录的 `resources/` 下。

## Resource
- ID `string` 资源唯一标识符
- Protocol `string` 来源协议（适配器 context ID）
- RefLink `string` 协议特定的资源引用链接
- FilePath `string` 资源文件本地路径
- Downloaded `time.Time` 资源记录时间

### grb.SaveResourceLink(contextID string, refLink string) string
保存资源引用链接，返回生成的资源 ID。此时并不会下载文件，而是等到 `LoadResourceFromID` 时再按需下载。

### grb.LoadResourceFromID(id string) (string, error)
根据资源 ID 获取本地文件路径。如果文件还没下载，会通过对应适配器的 `DownloadResourceFromRefLink` 下载并缓存。
