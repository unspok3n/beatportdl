package taglib

// #include <stdlib.h>
// #include <taglib/tag_c.h>
import "C"
import "unsafe"

func getCCharPointer(s string) *C.char {
	b := append([]byte(s), 0)
	return (*C.char)(C.CBytes(b))
}

func convertAndFree(cstr *C.char) string {
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr)
}

func toGoStringArray(cArray **C.char) []string {
	var goArray []string

	elem := cArray

	for elem != nil && *elem != nil {
		goArray = append(goArray, C.GoString(*elem))

		elem = (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(elem)) + unsafe.Sizeof(*elem)))
	}

	return goArray
}
