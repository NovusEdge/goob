//go:build !linux

package main

func getGlobalCursor() (x, y, buttons int) {
	// ponytail: windows/mac cursor tracking TBD
	return -1, -1, 0
}
