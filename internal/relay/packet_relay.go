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
	"context"
	"io"
	"log"
	"net"
	"sync"
)

const defaultBufferSize = 4096

// Type for buffer
type bufChan = chan []byte

// Packet relay struct
type packetRelay struct {
	wg *sync.WaitGroup
}

// Run single instance of packet relay.
// Create listener for incoming traffic, make new tcp connection to raddr and do relay traffic between them.
func (pry packetRelay) runRelay(ctx context.Context, local, remote string) {
	log.Printf("pkt_relay: start relaying between address %s <-> %s\n", local, remote)

	err := runListener(ctx, local, func(ctx context.Context, incomingConn net.Conn) {
		log.Printf("pkt_relay: prepare relaying to remote %s", remote)

		defer incomingConn.Close()

		outgoingConn, err := newOutgoingConn(remote)
		if err != nil {
			log.Printf("pkt_relay: failed to connect to remote %s err=%s", remote, err)
			return
		}

		defer outgoingConn.Close()

		pry.realyPackets(incomingConn, outgoingConn)
	})

	if err != nil {
		log.Fatalf("pkt_relay: failed to run relaying between address %s <-> %s\n", local, remote)
		return
	}

	log.Printf("pkt_relay: stop relaying between address %s <-> %s\n", local, remote)
}

// Bind incloming and outgoint connection via channels
func (pry packetRelay) realyPackets(in net.Conn, out net.Conn) {
	log.Printf("pkt_relay: relaying packets %s <-> %s", in.RemoteAddr(), out.RemoteAddr())

	ich, och := make(bufChan, 1), make(bufChan, 1)

	// run in -> och
	// 	   in <- ich
	pry.relay(in, ich, och)

	// run out -> och
	// 	   out <- ich
	pry.relay(out, och, ich)

	pry.wg.Wait()

	log.Printf("pkt_relay: finish relaying packets %s <-> %s", in.RemoteAddr(), out.RemoteAddr())
}

// Relay traffic from conn to wch and rch to conn
func (pry packetRelay) relay(conn net.Conn, rch bufChan, wch bufChan) {
	pry.wg.Add(1)
	go func() {
		connToChanRelay(conn, wch)

		log.Printf("pkt_relay: close conn(%s)->chan\n", conn.LocalAddr())

		close(wch)

		pry.wg.Done()
	}()

	pry.wg.Add(1)
	go func() {
		chanToConnRelay(rch, conn)

		log.Printf("pkt_relay: close chan->conn(%s)\n", conn.RemoteAddr())

		conn.Close()

		pry.wg.Done()
	}()
}

func connToChanRelay(conn net.Conn, ch bufChan) {
	log.Printf("pkt_relay: start conn(%s) ---> chan packets realy\n", conn.RemoteAddr())

	for {
		// TODO: rework with pool
		buf := make([]byte, defaultBufferSize)

		read, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("pkt_relay: conn(%s) err=%s faield to read from net", conn.RemoteAddr(), err)

				return
			}

			log.Printf("pkt_relay: conn(%s)->chan done relaying by EOF", conn.RemoteAddr())

			return
		}

		ch <- buf[0:read]
	}
}

func chanToConnRelay(ch bufChan, conn net.Conn) {
	log.Printf("pkt_relay: start chan ---> %s packets realy\n", conn.RemoteAddr())

	for {
		buf, ok := <-ch
		if !ok {
			log.Printf("pkt_relay: chan->conn(%s) complete relaying\n", conn.RemoteAddr())
			return
		}

		if _, err := conn.Write(buf); err != nil {
			log.Printf("pkt_relay: chan->conn(%s) fail to relay\n", conn.RemoteAddr())
			return
		}
	}
}

// Create new packets realy
func newPacketRelay() packetRelay {
	return packetRelay{
		wg: &sync.WaitGroup{},
	}
}
