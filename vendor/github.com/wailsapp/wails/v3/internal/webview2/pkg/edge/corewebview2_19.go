//go:build windows

package edge

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

const iCoreWebView2Version2Through18MethodCount = 58

type iCoreWebView2_19Vtbl struct {
	iCoreWebView2Vtbl
	version2Through18         [iCoreWebView2Version2Through18MethodCount]ComProc
	getMemoryUsageTargetLevel ComProc
	putMemoryUsageTargetLevel ComProc
}

type iCoreWebView2_19 struct {
	vtbl *iCoreWebView2_19Vtbl
}

var iCoreWebView2_19IID = windows.GUID{Data1: 0x6921f954, Data2: 0x79b0, Data3: 0x437f, Data4: [8]byte{0xa9, 0x97, 0xc8, 0x58, 0x11, 0x89, 0x7c, 0x68}}

func (i *ICoreWebView2) SetMemoryUsageTargetLevel(level uint32) error {
	if i == nil || i.vtbl == nil {
		return errors.New("WebView2 core is unavailable")
	}
	var target *iCoreWebView2_19
	hr, _, _ := i.vtbl.QueryInterface.Call(uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(&iCoreWebView2_19IID)), uintptr(unsafe.Pointer(&target)))
	if windows.Handle(hr) != windows.S_OK {
		return windows.Errno(hr)
	}
	if target == nil || target.vtbl == nil {
		return errors.New("WebView2 memory target interface is unavailable")
	}
	defer target.vtbl.Release.Call(uintptr(unsafe.Pointer(target)))
	hr, _, _ = target.vtbl.putMemoryUsageTargetLevel.Call(uintptr(unsafe.Pointer(target)), uintptr(level))
	if windows.Handle(hr) != windows.S_OK {
		return windows.Errno(hr)
	}
	return nil
}
