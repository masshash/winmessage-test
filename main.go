package main

import (
	"flag"
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32                   = syscall.MustLoadDLL("user32.dll")
	procEnumWindows          = user32.MustFindProc("EnumWindows")
	procGetWindowTextLengthW = user32.MustFindProc("GetWindowTextLengthW")
	procGetWindowTextW       = user32.MustFindProc("GetWindowTextW")
	procGetClassNameW        = user32.MustFindProc("GetClassNameW")
	procSendMessageW         = user32.MustFindProc("SendMessageW")
)

const (
	errnoERROR_IO_PENDING = 997
)

var (
	errERROR_IO_PENDING error = syscall.Errno(errnoERROR_IO_PENDING)
	errERROR_EINVAL     error = syscall.EINVAL
)

const (
	WM_QUERYENDSESSION = 0x11
	WM_ENDSESSION      = 0x16
	ENDSESSION_LOGOFF  = 0x80000000
)

func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return errERROR_EINVAL
	case errnoERROR_IO_PENDING:
		return errERROR_IO_PENDING
	}
	// TODO: add more here, after collecting data on the common
	// error values see on Windows. (perhaps when running
	// all.bat?)
	return e
}

func enumWindows(enumFunc uintptr, lparam uintptr) bool {
	res, _, _ := syscall.Syscall(procEnumWindows.Addr(), 2, enumFunc, lparam, 0)
	return res != 0
}

func getWindowTextW(hwnd HWND, str *uint16, maxCount uint32) (len int) {
	res, _, _ := syscall.Syscall(procGetWindowTextW.Addr(), 3, uintptr(hwnd), uintptr(unsafe.Pointer(str)), uintptr(maxCount))
	len = int(res)
	return
}

func getWindowTextLengthW(hwnd HWND) (len int) {
	res, _, _ := syscall.Syscall(procGetWindowTextLengthW.Addr(), 1, uintptr(hwnd), 0, 0)
	len = int(res)
	return
}

func getClassName(hWnd HWND, className *uint16, maxCount *uint32) int {
	res, _, err := syscall.Syscall(procGetClassNameW.Addr(), 3,
		uintptr(hWnd),
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(maxCount)))
	if res == 0 {
		fmt.Println(errnoErr(err))
		return 0
	}
	return int(res)
}

func sendMessageW(hWnd HWND, msg, wParam, lParam uint32) uintptr {
	ret, _, _ := syscall.Syscall6(procSendMessageW.Addr(), 4,
		uintptr(hWnd),
		uintptr(msg),
		uintptr(wParam),
		uintptr(lParam),
		0,
		0)
	return ret
}

func getWindowTitle(hwnd HWND) (title string) {
	titleLen := getWindowTextLengthW(hwnd) + 1
	windowTitleUInt16a := make([]uint16, titleLen)
	getWindowTextW(hwnd, &windowTitleUInt16a[0], uint32(titleLen))
	title = syscall.UTF16ToString(windowTitleUInt16a)
	return
}

func getWindowClassName(hwnd HWND) (className string) {
	var nameLen uint32 = 256
	classNameUInt16a := make([]uint16, nameLen)
	getClassName(hwnd, &classNameUInt16a[0], &nameLen)
	className = syscall.UTF16ToString(classNameUInt16a)
	return
}

type HWND uintptr

func main() {
	w := flag.Uint("w", 0, "window handle target")
	s := flag.Bool("s", false, "send WM_ENDSESSION")
	flag.Parse()

	target := HWND(*w)

	if target != 0 {
		retmsg := sendMessageW(target, WM_QUERYENDSESSION, 0, 0)
		if retmsg == 1 {
			fmt.Println("returned TRUE")
			if *s {
				sendMessageW(target, WM_ENDSESSION, 1, 0)
			}
		} else {
			fmt.Println("returned FALSE")
		}
		return
	}

	classNameLen := uint32(syscall.MAX_LONG_PATH)
	classNameUInt16a := make([]uint16, classNameLen)
	enumFunc := syscall.NewCallback(func(hwnd HWND, _ uintptr) uintptr {
		title := getWindowTitle(hwnd)
		getClassName(hwnd, &classNameUInt16a[0], &classNameLen)
		className := syscall.UTF16ToString(classNameUInt16a)
		fmt.Printf("%d\t\"%s\"\t\"%s\"\n", hwnd, title, className)
		return 1
	})
	enumWindows(enumFunc, 0)
}
