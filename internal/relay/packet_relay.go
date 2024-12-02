/**
 *
 * MIT License
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
	"io"
	"log"
)

const defaultBufferSize = 4096

// Buffer channel
type bufChan = chan []byte

// RO only channel
type roBufChan = <-chan []byte

// WO onlu channel
type woBufChan = chan<- []byte

func connToChanRelay(conn io.Reader, ch woBufChan, raddr string) {
	log.Printf("pkt_relay: start conn(%s) ---> chan packets realy\n", raddr)

	for {
		// TODO: rework with pool
		buf := make([]byte, defaultBufferSize)

		read, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("pkt_relay: conn(%s) err=%s faield to read from net", raddr, err)
				return
			}

			log.Printf("pkt_relay: conn(%s)->chan done relaying by EOF", raddr)

			return
		}

		ch <- buf[0:read]
	}
}

func chanToConnRelay(conn io.Writer, ch roBufChan, raddr string) {
	log.Printf("pkt_relay: start chan ---> %s packets realy\n", raddr)

	for {
		buf, ok := <-ch
		if !ok {
			log.Printf("pkt_relay: chan->conn(%s) complete relaying\n", raddr)
			return
		}

		if _, err := conn.Write(buf); err != nil {
			log.Printf("pkt_relay: chan->conn(%s) fail to relay err=%s\n", raddr, err)
			return
		}
	}
}
