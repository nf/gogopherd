package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"os"
	"log"
	"net"
)

var (
	host = flag.String("host", "localhost", "server hostname")
	port = flag.Int("port", 70, "server port number")
	root = flag.String("root", "", "gopher content root")
)

const (
	T_PlainText = '0'
	T_Directory = '1'
	T_Error     = '3'
	T_Binary    = '9'
	T_GIF       = 'g'
	T_HTML      = 'h'
	T_Info      = 'i'
	T_Image     = 'I'
	T_Audio     = 's'
	T_Sentinel  = 0
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
		if e.Type == T_Sentinel {
			continue
		}
		fmt.Fprint(&b, e)
	}
	fmt.Fprint(&b, ".\r\n")
	return b.String()
}

func (l *Listing) VisitDir(path string, f *os.FileInfo) bool {
	if len(*l) == 0 {
		*l = append(*l, Entry{Type: T_Sentinel})
		return true
	}
	*l = append(*l, Entry{T_Directory, f.Name, path[len(*root):], *host, *port})
	return false
}

func (l *Listing) VisitFile(path string, f *os.FileInfo) {
	*l = append(*l, Entry{T_Binary, f.Name, path[len(*root):], *host, *port})
}

func Serve(c net.Conn) {
	defer c.Close()
	var p string
	n, err := fmt.Fscanln(c, &p)
	if n != 1 || err != nil {
		fmt.Fprint(c, Error("invalid request"))
		return
	}
	filename := *root + filepath.Clean("/" + p)
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
	f, err := os.Open(filename, os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprint(c, Error("couldn't open file"))
		return
	}
	io.Copy(c, f)
}

func Error(msg string) Listing {
	return Listing{Entry{Type: T_Error, Display: msg}}
}

func main() {
	flag.Parse()
	if *root == "" {
		log.Fatal("Please specify a content root with -root")
	}
	if (*root)[len(*root)-1:] == "/" {
		*root = (*root)[:len(*root)-1]
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
