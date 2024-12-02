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
	"net/netip"
	"sync"
)

// Config interface
type Config interface {
	Local() netip.Addr
	Remote() netip.Addr
	Ports() []uint16
}

// Packet relay struct
type packetRelay struct {
	wg *sync.WaitGroup
}

// Create and run relay on ports based on provided config
func Run(ctx context.Context, cfg Config) {
	log.Printf("relay: run relay with config %s", cfg)

	wg := sync.WaitGroup{}

	for _, port := range cfg.Ports() {
		pry := newPacketRelay()

		local, remote := makeAddr(cfg.Local(), port), makeAddr(cfg.Remote(), port)

		wg.Add(1)

		go func() {
			pry.runRelay(ctx, local, remote)

			wg.Done()
		}()
	}

	wg.Wait()

	log.Println("relay: stop relay")
}

// Run single instance of packet relay.
// Create listener for incoming traffic, make new tcp connection to raddr and do relay traffic between them.
func (pry packetRelay) runRelay(ctx context.Context, local, remote string) {
	log.Printf("pkt_relay: start relaying between address %s <-> %s\n", local, remote)

	err := listenConn(ctx, local, func(ctx context.Context, inConn net.Conn) {
		log.Printf("pkt_relay: prepare relaying to remote %s", remote)

		defer inConn.Close()

		outConn, err := newOutgoingConn(remote)
		if err != nil {
			log.Printf("pkt_relay: failed to connect to remote %s err=%s", remote, err)
			return
		}

		defer outConn.Close()

		wg := &sync.WaitGroup{}

		lctx, cancel := context.WithCancel(ctx)

		wg.Add(1)
		go func() {
			<-lctx.Done()

			defer inConn.Close()

			defer outConn.Close()

			wg.Done()
		}()

		pry.realyPackets(inConn, inConn.RemoteAddr().String(), outConn, outConn.RemoteAddr().String())

		cancel()

		wg.Wait()
	})

	if err != nil {
		log.Printf("pkt_relay: failed to run relaying between address %s <-> %s err=%s\n", local, remote, err)
		return
	}

	log.Printf("pkt_relay: stop relaying between address %s <-> %s\n", local, remote)
}

// Bind incoming and outgoing connection via channels
func (pry packetRelay) realyPackets(in io.ReadWriteCloser, inRAddr string, out io.ReadWriteCloser, outRAddr string) {
	log.Printf("pkt_relay: relaying packets %s <-> %s", inRAddr, outRAddr)

	ich, och := make(bufChan, 1), make(bufChan, 1)

	// run in -> och
	//     in <- ich
	pry.relay(in, inRAddr, ich, och)

	// run out -> och
	//     out <- ich
	pry.relay(out, outRAddr, och, ich)

	// wait for all 4 relay routines stops
	pry.wg.Wait()

	log.Printf("pkt_relay: finish relaying packets %s <-> %s", inRAddr, outRAddr)
}

// Relay traffic from conn to wch and rch to conn
func (pry packetRelay) relay(conn io.ReadWriteCloser, raddr string, rch roBufChan, wch woBufChan) {
	pry.wg.Add(1)
	go func() {
		connToChanRelay(conn, wch, raddr)

		log.Printf("pkt_relay: close conn(%s)->chan\n", raddr)

		close(wch)

		pry.wg.Done()
	}()

	pry.wg.Add(1)
	go func() {
		chanToConnRelay(conn, rch, raddr)

		log.Printf("pkt_relay: close chan->conn(%s)\n", raddr)

		conn.Close()

		pry.wg.Done()
	}()
}

// Create new packets realy
func newPacketRelay() packetRelay {
	return packetRelay{wg: &sync.WaitGroup{}}
}

// Make valid string address from addr + port
func makeAddr(addr netip.Addr, port uint16) string {
	return netip.AddrPortFrom(addr, port).String()
}
