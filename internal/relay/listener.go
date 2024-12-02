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
	"errors"
	"log"
	"net"
)

// Handle new connection on go routine
type acceptorFunc func(ctx context.Context, conn net.Conn)

// Run listener bound to addr and call connHandler on new incoming connection
func runListener(ctx context.Context, addr string, connHandler acceptorFunc) error {
	log.Printf("relay: start listening on addr=%s\n", addr)

	lc := &net.ListenConfig{}

	listener, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		log.Printf("relay: failed to listen addr=%s, err=%s\n", addr, err)
		return errors.Join(ErrListenAddr, err)
	}

	go func() {
		<-ctx.Done()

		log.Printf("relay: close relay listener addr=%s\n", addr)

		if err := listener.Close(); err != nil {
			log.Println("relay: failed to close listener")
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("relay: failed to accept for addr=%s err=%v\n", addr, err)
			break
		}

		log.Printf("relay: accept connection on addr=%s\n", addr)

		go connHandler(ctx, conn)
	}

	return nil
}
