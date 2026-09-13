package bot_context

// Feature 定义适配器所支持的能力特性
type Feature string

const (
	FeatureText           Feature = "text"
	FeatureImage          Feature = "image"
	FeatureMarkdown       Feature = "markdown"
	FeatureInlineImage    Feature = "inline_image"
	FeatureVoice          Feature = "voice"
	FeatureFile           Feature = "file"
	FeatureInlineKeyboard Feature = "inline_keyboard"
)

// FeatureProvider 接口表示提供特性支持查询的对象
type FeatureProvider interface {
	// Features 返回当前适配器所支持的所有特性列表
	Features() []Feature
	// SupportsFeature 判断当前适配器是否支持某个特性
	SupportsFeature(feature Feature) bool
}
