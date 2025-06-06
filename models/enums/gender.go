package enums

// Gender 表示性别的枚举类型
type Gender uint

const (
	Unknown Gender = 0 // 未知
	Male    Gender = 1 // 男性
	Female  Gender = 2 // 女性
)
