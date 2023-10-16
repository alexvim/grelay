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
)

// Nif ...
type Nif struct {
	remoteAddress string
	remotrPort    uint16

	inboundDataPort  net.Conn
	outboundDataPort net.Conn
}

// MakeNif ...
func MakeNif(conn net.Conn, ra string, rp uint16) *Nif {
	nif := new(Nif)
	nif.inboundDataPort = conn
	nif.remoteAddress = ra
	nif.remotrPort = rp
	return nif
}

// Prepare ...
func (n *Nif) Prepare() (string, uint16, error) {

	var rfa string = fmt.Sprintf("%s:%d", n.remoteAddress, n.remotrPort)
	var err error = nil

	fmt.Printf("nif: open remote data port to adds=%s\n", rfa)

	var connType string = "tcp4"
	if GetTcpAddrType(n.inboundDataPort.LocalAddr()) == AddrTypeIpv6 {
		connType = "tcp6"
	}

	n.outboundDataPort, err = net.Dial(connType, rfa)
	if err != nil {
		n.inboundDataPort.Close()
		fmt.Printf("nif: failed to connect to remote for adds=%s error=%s\n ", rfa, err.Error())
		return "", 0, err
	}

	ip := n.outboundDataPort.LocalAddr().(*net.TCPAddr)

	return ip.IP.String(), uint16(ip.Port), nil
}

// Run ...
func (n *Nif) Run() {

	fmt.Println("nif: start relaying")

	inboundRelay := makeRelay(n.inboundDataPort, n.outboundDataPort)
	outboundRelay := makeRelay(n.outboundDataPort, n.inboundDataPort)

	done := make(chan bool)

	// wait for one of relay part is done. This means one part of relay is disconnected
	// and the other one could be closed
	go inboundRelay.run(done)
	go outboundRelay.run(done)

	// wait for someone done their task
	<-done

	fmt.Println("nif: stopping relay")

	if inboundRelay.done {
		outboundRelay.src.Close()
	} else {
		inboundRelay.src.Close()
	}

	<-done

	n.inboundDataPort.Close()
	n.outboundDataPort.Close()

	n.inboundDataPort = nil
	n.outboundDataPort = nil

	fmt.Println("nif: stop relay")
}
