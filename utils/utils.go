package utils

func StrVal(v interface{}) string {
	switch s := v.(type) {
	case *string:
		if s == nil {
			return ""
		}
		return *s
	case string:
		return s
	default:
		return ""
	}
}
