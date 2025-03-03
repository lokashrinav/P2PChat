package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type Msg struct {
	R string `json:"r"`
	S string `json:"s"`
	M string `json:"m"`
	T int64  `json:"t"`
}

var (
	rm = make(map[string]bool)
	cr string
	mu sync.RWMutex
	ps []string
	u  string
)

func main() {
	p := flag.String("port", "8000", "")
	peers := flag.String("peers", "", "")
	usr := flag.String("username", "anon", "")
	flag.Parse()
	u = *usr
	if *peers != "" {
		ps = strings.Split(*peers, ",")
	}
	go serv(*p)
	fmt.Println("Chat on port", *p)
	reader := bufio.NewReader(os.Stdin)
	for {
		inp, _ := reader.ReadString('\n')
		inp = strings.TrimSpace(inp)
		if inp == "" {
			continue
		}
		if inp[0] == '/' {
			cmd(inp)
		} else {
			send(inp)
		}
	}
}

func cmd(s string) {
	sp := strings.SplitN(s, " ", 2)
	if sp[0] == "/join" {
		mu.Lock()
		rm[sp[1]] = true
		cr = sp[1]
		mu.Unlock()
		fmt.Println("Joined", sp[1])
	} else if sp[0] == "/switch" {
		mu.RLock()
		if rm[sp[1]] {
			cr = sp[1]
			fmt.Println("Switched to", sp[1])
		} else {
			fmt.Println("Not in", sp[1])
		}
		mu.RUnlock()
	} else if sp[0] == "/rooms" {
		mu.RLock()
		if len(rm) == 0 {
			fmt.Println("No rooms")
		} else {
			for r := range rm {
				if r == cr {
					fmt.Println("*", r, "(active)")
				} else {
					fmt.Println("*", r)
				}
			}
		}
		mu.RUnlock()
	} else if sp[0] == "/leave" {
		mu.Lock()
		delete(rm, sp[1])
		if cr == sp[1] {
			cr = ""
		}
		mu.Unlock()
		fmt.Println("Left", sp[1])
	} else {
		fmt.Println("Unknown cmd")
	}
}

func send(txt string) {
	mu.RLock()
	r := cr
	mu.RUnlock()
	if r == "" {
		fmt.Println("Join a room first")
		return
	}
	m := Msg{R: r, S: u, M: txt, T: time.Now().Unix()}
	d, _ := json.Marshal(m)
	fmt.Printf("[%s] %s: %s\n", r, u, txt)
	for _, a := range ps {
		go func(a string) {
			c, _ := net.Dial("tcp", a)
			c.Write(d)
			c.Close()
		}(a)
	}
}

func serv(p string) {
	ln, _ := net.Listen("tcp", ":"+p)
	for {
		c, _ := ln.Accept()
		go hand(c)
	}
}

func hand(c net.Conn) {
	defer c.Close()
	dec := json.NewDecoder(c)
	var m Msg
	dec.Decode(&m)
	mu.RLock()
	j := rm[m.R]
	mu.RUnlock()
	if !j {
		fmt.Printf("[Room %s (not joined)] %s: %s\n", m.R, m.S, m.M)
	} else {
		fmt.Printf("[%s] %s: %s\n", m.R, m.S, m.M)
	}
}
