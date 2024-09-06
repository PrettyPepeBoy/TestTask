package cast

import "unsafe"

func ByteArrayToSting(arr []byte) string {
	return unsafe.String(unsafe.SliceData(arr), len(arr))
}

func StringToByteArray(str string) []byte {
	return unsafe.Slice(unsafe.StringData(str), len(str))
}
