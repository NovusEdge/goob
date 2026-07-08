//go:build !linux

package main

func getGlobalCursor() (int, int) {
	// ponytail: windows/mac cursor tracking TBD
	return -1, -1
}
