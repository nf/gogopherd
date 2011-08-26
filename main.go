package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

var (
	host = flag.String("host", "localhost", "server hostname")
	port = flag.Int("port", 70, "server port number")
	root string
)

type Entry struct {
	Type     byte
	Display  string
	Selector string
	Hostname string
	Port     int
}

func (e Entry) String() string {
	return fmt.Sprintf("%c%s\t%s\t%s\t%d\r\n",
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

func (l *Listing) VisitDir(path string, f *os.FileInfo) bool {
	if len(*l) == 0 {
		*l = append(*l, Entry{}) // sentinel value
		return true
	}
	*l = append(*l, Entry{'1', f.Name, path[len(root):], *host, *port})
	return false
}

var suffixes = map[string]byte{
	"aiff": 's',
	"au": 's',
	"gif": 'g',
	"go": '0',
	"html": 'h',
	"jpeg": 'I',
	"jpg": 'I',
	"mp3": 's',
	"png": 'I',
	"txt": '0',
	"wav": 's',
}

func (l *Listing) VisitFile(path string, f *os.FileInfo) {
	t := byte('9') // Binary
	for s, c := range suffixes {
		if strings.HasSuffix(path, "."+s) {
			t = c
			break
		}
	}
	*l = append(*l, Entry{t, f.Name, path[len(root):], *host, *port})
}

func Serve(c net.Conn) {
	defer c.Close()
	var p string
	n, err := fmt.Fscanln(c, &p)
	if n != 1 || err != nil {
		fmt.Fprint(c, Error("invalid request"))
		return
	}
	filename := root + filepath.Clean("/"+p)
	fi, err := os.Stat(filename)
	if err != nil {
		fmt.Fprint(c, Error("not found"))
		return
	}
	if fi.IsDirectory() {
		var list Listing
		filepath.Walk(filename, &list, nil)
		fmt.Fprint(c, list)
		return
	}
	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprint(c, Error("couldn't open file"))
		return
	}
	io.Copy(c, f)
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
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *host, *port))
	if err != nil {
		log.Fatal(err)
	}
	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go Serve(c)
	}
}
