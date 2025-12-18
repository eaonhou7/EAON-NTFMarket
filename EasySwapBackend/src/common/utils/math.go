package utils

// Min 返回两个整数中的较小值
func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

// Max 返回两个 int64 整数中的较大值
func Max(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
}
