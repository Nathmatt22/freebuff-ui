//go:build !linux

package main

// afterInstallNote has nothing to add outside Linux: an installed copy is
// simply launched the usual way.
func afterInstallNote() string { return "" }