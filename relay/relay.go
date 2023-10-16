/**
 *
 * MIT License
 *
 * Copyright (c) 2023 Alexander Morozov
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 *
 */

package relay

import (
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	queueLength int = 1000
	readBlock   int = 8192
)

var relayGlobalCounter uint64 = 0

type Relay struct {
	id        uint64
	src       net.Conn
	dst       net.Conn
	readEOF   bool
	writeEOF  bool
	done      bool
	relayChan chan *[]byte
	sch       chan<- bool
}

func makeRelay(src net.Conn, dst net.Conn) *Relay {
	relayGlobalCounter++
	r := new(Relay)
	r.id = relayGlobalCounter
	r.src = src
	r.dst = dst
	r.readEOF = false
	r.writeEOF = false
	r.done = false
	r.relayChan = make(chan *[]byte, queueLength)
	return r
}

func (r *Relay) close() {
	close(r.relayChan)
	close(r.sch)
}

func (r *Relay) run(sch chan<- bool) {

	var wg sync.WaitGroup

	wg.Add(2)

	go r.read(r.src, &wg)
	go r.write(r.dst, &wg)

	wg.Wait()

	r.done = true
	sch <- true
}

func (r *Relay) read(conn net.Conn, wg *sync.WaitGroup) {

	defer wg.Done()

	var ch chan<- *[]byte = r.relayChan

	remoteAddr := conn.LocalAddr().String()

	fmt.Printf("relay{%d}: start read stream from: %s\n", r.id, remoteAddr)

	bufferLen := readBlock * (cap(ch) + 100)
	buf := make([]byte, bufferLen)
	rindex := 0
	for {

		if r.writeEOF {
			fmt.Printf("relay{%d}: error {write stream closed} on reading from %s\n", r.id, remoteAddr)
			break
		}

		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		byteRead, err := conn.Read(buf[rindex : rindex+readBlock])
		if e, ok := err.(net.Error); ok && e.Timeout() {
			continue
		} else if err != nil {
			fmt.Printf("relay{%d}: error {%s} on reading from %s\n", r.id, err.Error(), remoteAddr)
			// send EOF to write
			ch <- nil
			break
		}

		if byteRead <= 0 {
			fmt.Printf("relay{%d}: read again on %s\n", r.id, remoteAddr)
			continue
		}

		b := buf[rindex : rindex+byteRead]
		ch <- &b

		rindex = rindex + byteRead
		if rindex+readBlock > bufferLen {
			rindex = 0
		}
	}

	r.readEOF = true
	fmt.Printf("relay{%d}: close read stream from: %s\n", r.id, remoteAddr)
}

func (r *Relay) write(conn net.Conn, wg *sync.WaitGroup) {

	defer wg.Done()

	var ch <-chan *[]byte = r.relayChan

	remoteAddr := conn.RemoteAddr().String()

	fmt.Printf("relay{%d}: start write stream to %s\n", r.id, remoteAddr)

	for buf := <-ch; buf != nil; buf = <-ch {
		// TODO: Write uses async aproach, so buf passed to it shall not be altered in some period of time, but read may works
		// faster than write this force to buffer overwite and write data corruption
		if n, err := conn.Write(*buf); err != nil {
			fmt.Printf("relay{%d}: error {%s} on writing to %s\n", r.id, err.Error(), remoteAddr)
			break
		} else if n > 0 && n < len(*buf) {
			fmt.Printf("relay{%d}: error {unable to write full buffer n=%d against buf=%d} on writing to %s\n", r.id, n, len(*buf), remoteAddr)
			break
		}
	}

	r.writeEOF = true
	// make fake read to unqueue data and unclock write to channel if it was full
	if len(ch) > 0 {
		fmt.Printf("relay{%d}: error {queue is not drain} while writing to %s, do fake dequeue\n", r.id, remoteAddr)
		<-ch
	}
	fmt.Printf("relay{%d}: close write stream to %s\n", r.id, remoteAddr)
}
