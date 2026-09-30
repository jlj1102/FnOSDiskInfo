//go:build !linux

package main

// nvmeIdentify is Linux-only (ioctl); the Windows dev build returns nil and
// the smartctl JSON path stays in charge.
func nvmeIdentify(string) []byte { return nil }
