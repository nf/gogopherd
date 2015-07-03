// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package main

import (
	"fmt"
	"net"
	"os"
	"syscall"
)

func sendfile(c *net.TCPConn, f *os.File, fi os.FileInfo) {
	sockFile, err := c.File()
	if err != nil {
		fmt.Fprint(c, Error(fmt.Sprintf("couldn't get file sock: %x", err)))
	}
	syscall.Sendfile(int(sockFile.Fd()), int(f.Fd()), nil, int(fi.Size()))
}
