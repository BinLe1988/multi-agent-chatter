package filter

// ContentType 定义内容类型
type ContentType byte

const (
	ContentTypeText  ContentType = 0
	ContentTypeImage ContentType = 1
	ContentTypeAudio ContentType = 2
	ContentTypeVideo ContentType = 3
)
