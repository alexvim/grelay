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

package main

import (
	"flag"
	"fmt"
	"grelay/relay"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

func main() {

	var localAddress string
	var remoteAddress string
	var portsArg string

	flag.StringVar(&localAddress, "l", "", "local ipv4 address where incoming traffic is come i.e. one of addresses on transitional host which is visible for target host/application")
	flag.StringVar(&remoteAddress, "r", "", "remote address somethere in target vpn/subnet/tunnel")
	flag.StringVar(&portsArg, "p", "", "comma separated port list to be forwarded")

	flag.Parse()

	var ports []int
	for _, p := range strings.Split(portsArg, ",") {
		// string to int
		port, err := strconv.Atoi(p)
		if err != nil {
			panic(err)
		}
		ports = append(ports, port)
	}

	fmt.Printf("main: l=%s, r=%s, p=%v\n", localAddress, remoteAddress, ports)

	for _, port := range ports {

		// accept connection for port and forward data to remote
		go func(laddr string, raddr string, port uint16) {

			addr := laddr + ":" + strconv.Itoa(int(port))

			fmt.Printf("main: start listening on address %s\n", addr)

			listener, err := net.Listen("tcp", addr)
			if err != nil {
				fmt.Printf("main: failed to listern %s port err={%s}", addr, err.Error())
				panic(err)
			}

			for {
				connection, err := listener.Accept()
				if err != nil {
					fmt.Printf("main: failed to accept connection on %s port err={%s}", listener.Addr().String(), err.Error())
					panic(err)
				}

				go func(connection net.Conn, laddr string, raddr string, port uint16) {
					fmt.Printf("main: incoming connection on address %s\n", laddr)

					nif := relay.MakeNif(connection, raddr, port)

					nif.Prepare()

					nif.Run()

					fmt.Printf("main: close connection on address %s\n", laddr)

				}(connection, addr, raddr, port)
			}

		}(localAddress, remoteAddress, uint16(port))
	}

	done := make(chan os.Signal, 1)

	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Press ctrl+c to stop application")

	// Will block here until user hits ctrl+c
	<-done
}
