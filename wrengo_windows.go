package wrengo

import "syscall"

func loadLibrary(name string) (uintptr, error) {
	handle, err := syscall.LoadLibrary(name)
	return uintptr(handle), err
}
