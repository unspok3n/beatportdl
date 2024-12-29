package taglib

/*
	#cgo LDFLAGS: -ltag -ltag_c -lz
	#include <stdlib.h>
	#include <taglib/tag_c.h>
	#include "extensions.h"
*/
import "C"

import (
	"errors"
	"unsafe"
)

var (
	ErrInvalid   = errors.New("invalid file")
	ErrStripMp4  = errors.New("cannot strip mp4 tags")
	ErrSave      = errors.New("cannot save file")
	ErrNoPicture = errors.New("no picture")
)

func init() {
	C.taglib_set_string_management_enabled(0)
	C.taglib_id3v2_set_default_text_encoding(3)
}

// File API
type File struct {
	fp    *C.TagLib_File
	tag   *C.TagLib_Tag
	props *C.TagLib_AudioProperties
}

func Read(filename string) (*File, error) {
	cs := C.CString(filename)
	defer C.free(unsafe.Pointer(cs))
	fp := C.taglib_file_new_wide(cs)
	if fp == nil || C.taglib_file_is_valid(fp) == 0 {
		return nil, ErrInvalid
	}
	return &File{
		fp:    fp,
		tag:   C.taglib_file_tag(fp),
		props: C.taglib_file_audioproperties(fp),
	}, nil
}

func (f *File) Close() {
	C.taglib_file_free(f.fp)
	f.fp = nil
	f.tag = nil
	f.props = nil
}

func (f *File) Save() error {
	if C.taglib_file_save(f.fp) != 1 {
		return ErrSave
	}
	return nil
}

func (f *File) SetItemMp4(key, value string) {
	keyC := C.CString(key)
	defer C.free(unsafe.Pointer(keyC))
	valueC := C.CString(value)
	defer C.free(unsafe.Pointer(valueC))
	C.taglib_set_item_mp4(f.fp, keyC, valueC)
}

func (f *File) StripMp4() error {
	valueC := C.taglib_strip_mp4(f.fp)
	if int(valueC) != 1 {
		return ErrStripMp4
	}
	return nil
}

// Properties API
func (f *File) GetProperty(property string) string {
	propertyC := C.CString(property)
	defer C.free(unsafe.Pointer(propertyC))
	valueC := C.taglib_property_get(f.fp, propertyC)
	defer C.free(unsafe.Pointer(valueC))
	value := C.GoString(*valueC)
	return value
}

func (f *File) SetProperty(property string, value *string) {
	propertyC := getCCharPointer(property)
	defer C.free(unsafe.Pointer(propertyC))
	var valueC *C.char
	if value != nil {
		valueC = getCCharPointer(*value)
		defer C.free(unsafe.Pointer(valueC))
	}
	C.taglib_property_set(f.fp, propertyC, valueC)
}

func (f *File) PropertyKeys() ([]string, error) {
	keysC := C.taglib_property_keys(f.fp)
	defer C.taglib_property_free(keysC)
	var keys []string
	if keysC == nil {
		return keys, nil
	}
	for i := 0; ; i++ {
		cstr := (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(keysC)) + uintptr(i)*unsafe.Sizeof(uintptr(0))))
		if *cstr == nil {
			break
		}
		keys = append(keys, C.GoString(*cstr))
	}
	return keys, nil
}

func (f *File) SampleRate() int {
	return int(C.taglib_audioproperties_samplerate(f.props))
}

// Complex Properties API
type Picture struct {
	MimeType    string
	PictureType string
	Description string
	Data        []byte
	Size        uint
}

func (f *File) GetPicture() (*Picture, error) {
	cs := C.CString("PICTURE")
	defer C.free(unsafe.Pointer(cs))
	property := C.taglib_complex_property_get(f.fp, cs)
	if property == nil || *property == nil {
		return nil, ErrNoPicture
	}
	defer C.taglib_complex_property_free(property)
	var data C.TagLib_Complex_Property_Picture_Data
	C.taglib_picture_from_complex_property(property, &data)

	return &Picture{
		MimeType:    C.GoString(data.mimeType),
		PictureType: C.GoString(data.pictureType),
		Description: C.GoString(data.description),
		Data:        C.GoBytes(unsafe.Pointer(data.data), C.int(data.size)),
		Size:        uint(data.size),
	}, nil
}

func (f *File) SetPicture(picture *Picture) error {
	dataC := C.CBytes(picture.Data)
	defer C.free(dataC)
	descC := C.CString(picture.Description)
	defer C.free(unsafe.Pointer(descC))
	mimeC := C.CString(picture.MimeType)
	defer C.free(unsafe.Pointer(mimeC))
	typeC := C.CString(picture.PictureType)
	defer C.free(unsafe.Pointer(typeC))

	C.taglib_set_picture(f.fp, (*C.char)(dataC), C.uint(picture.Size), descC, mimeC, typeC)
	return nil
}

func (f *File) ComplexPropertyKeys() ([]string, error) {
	keysC := C.taglib_complex_property_keys(f.fp)
	defer C.taglib_complex_property_free_keys(keysC)
	var keys []string
	if keysC == nil {
		return keys, nil
	}
	for i := 0; ; i++ {
		cstr := (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(keysC)) + uintptr(i)*unsafe.Sizeof(uintptr(0))))
		if *cstr == nil {
			break
		}
		keys = append(keys, C.GoString(*cstr))
	}
	return keys, nil
}
