// +build !darwin,!dragonfly,!freebsd,!linux,!netbsd,!openbsd,!solaris

package main

import (
	"io"
	"net"
	"os"
)

func sendfile(c *net.TCPConn, f *os.File, fi os.FileInfo) {
	io.Copy(c, f)
}
