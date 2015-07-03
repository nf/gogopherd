package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

var (
	host    = flag.String("host", "localhost", "hostname used in links")
	address = flag.String("address", "localhost", "listen on address")
	port    = flag.String("port", "70", "listen on port")
	root    string
)

type Entry struct {
	Type     byte
	Display  string
	Selector string
	Hostname string
	Port     string
}

func (e Entry) String() string {
	return fmt.Sprintf("%c%s\t%s\t%s\t%s\r\n",
		e.Type, e.Display, e.Selector, e.Hostname, e.Port)
}

type Listing []Entry

func (l Listing) String() string {
	var b bytes.Buffer
	for _, e := range l {
		if e.Type == 0 {
			continue // skip sentinel value
		}
		fmt.Fprint(&b, e)
	}
	fmt.Fprint(&b, ".\r\n")
	return b.String()
}

func (l Listing) VisitDir(path string, f os.FileInfo) error {
	if len(l) == 0 {
		l = append(l, Entry{}) // sentinel value
		return nil
	}
	l = append(l, Entry{'1', f.Name(), path[len(root)-1:], *host, *port})
	return filepath.SkipDir
}

var suffixes = map[string]byte{
	"aiff": 's',
	"au":   's',
	"gif":  'g',
	"go":   '0',
	"html": 'h',
	"jpeg": 'I',
	"jpg":  'I',
	"mp3":  's',
	"png":  'I',
	"txt":  '0',
	"wav":  's',
}

func (l Listing) VisitFile(path string, f os.FileInfo) {
	t := byte('9') // Binary
	for s, c := range suffixes {
		if strings.HasSuffix(path, "."+s) {
			t = c
			break
		}
	}
	l = append(l, Entry{t, f.Name(), path[len(root)-1:], *host, *port})
}

func Serve(c *net.TCPConn) {
	defer c.Close()
	connbuf := bufio.NewReader(c)
	p, _, err := connbuf.ReadLine()
	if err != nil {
		fmt.Fprint(c, Error("invalid request"))
		return
	}
	filename := root + filepath.Clean("/"+string(p))
	fi, err := os.Stat(filename)
	if err != nil {
		fmt.Fprint(c, Error("not found"))
		return
	}
	if fi.IsDir() {
		var list Listing
		walkFn := func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return list.VisitDir(path, info)
			}

			list.VisitFile(path, info)
			return nil
		}

		filepath.Walk(filename, walkFn)
		fmt.Fprint(c, list)
		return
	}
	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprint(c, Error("couldn't open file"))
		return
	}
	sendfile(c, f, fi)
}

func Error(msg string) Listing {
	return Listing{Entry{Type: 3, Display: msg}}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s directory\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(2)
	}

	flag.Parse()
	if root = flag.Arg(0); root == "" {
		flag.Usage()
	}
	if strings.HasSuffix(root, "/") {
		root = root[:len(root)-1]
	}
	listenAddr := net.JoinHostPort(*address, *port)
	l, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal(err)
	}

	for {
		c, err := l.Accept()
		tcpConn := c.(*net.TCPConn)
		if err != nil {
			log.Fatal(err)
		}
		go Serve(tcpConn)
	}
}
