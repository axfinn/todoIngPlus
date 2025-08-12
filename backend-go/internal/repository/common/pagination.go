package common

// PaginationParams 通用分页参数
// Page 从 1 开始; Limit >0
// 返回 skip/limit 供上层使用
func Normalize(page, limit, max int64) (int64, int64) {
	if limit <= 0 {
		limit = 20
	}
	if max > 0 && limit > max {
		limit = max
	}
	if page <= 0 {
		page = 1
	}
	return page, limit
}
