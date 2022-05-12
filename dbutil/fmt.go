package dbutil

// BoolToBitStr formats boolean as "0" or "1"
func BoolToBitStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
