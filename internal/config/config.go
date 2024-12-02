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
package config

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net/netip"
	"strconv"
	"strings"
)

const (
	locaParamlDesc  = "local ipv4 address where incoming traffic comes from i.e. one of addresses on transitional host which is visible for target host/application"
	remoteParamDesc = "remote address somethere in target vpn/subnet/tunnel"
	portParamDesc   = "comma separated port list to be forwarded"
)

// Configuration of this service
type Config struct {
	// Local address to bind and recevie data
	localAddress netip.Addr
	// Remote address where some peer are available
	remoteAddress netip.Addr
	// Ports to be forwarded
	ports []uint16
}

// Create new config based on args passed to app
//
// Example -l 127.0.0.1 -r 10.12.112.10 -p 1010,1080,443
func NewConfigFromCmdLineArgs(args []string) (Config, error) {
	log.Printf("config: parse agrs %v", args)

	var localArg string
	var remoteArg string
	var portsArg string

	flags := flag.NewFlagSet("", flag.ContinueOnError)

	flags.StringVar(&localArg, "l", "", locaParamlDesc)
	flags.StringVar(&remoteArg, "r", "", remoteParamDesc)
	flags.StringVar(&portsArg, "p", "", portParamDesc)

	if err := flags.Parse(args); err != nil {
		log.Printf("filaed to parse parameters err=%s", err)
		return Config{}, ErrInvalidArgs
	}

	return makeConfig(localArg, remoteArg, portsArg)
}

func (cfg Config) Local() netip.Addr {
	return cfg.localAddress
}

func (cfg Config) Remote() netip.Addr {
	return cfg.remoteAddress
}

func (cfg Config) Ports() []uint16 {
	return cfg.ports
}

func (cfg Config) String() string {
	return fmt.Sprintf("{%s -> %s for ports %v}", cfg.localAddress, cfg.remoteAddress, cfg.ports)
}

func makeConfig(localArg, remoteArg, portsArg string) (Config, error) {
	log.Printf("create new config local=%s, remote=%s, ports=%s\n", localArg, remoteArg, portsArg)

	lip, err := netip.ParseAddr(localArg)
	if err != nil {
		log.Printf("parameter %s is not valid ip address err=%s\n", localArg, err)
		return Config{}, ErrInvalidParameter
	}

	rip, err := netip.ParseAddr(remoteArg)
	if err != nil {
		log.Printf("parameter %s is not valid ip address err=%s\n", remoteArg, err)
		return Config{}, ErrInvalidParameter
	}

	ports := strings.Split(portsArg, ",")
	ipPorts := make([]uint16, 0, len(ports))

	for _, port := range ports {
		ipPort, err := strconv.Atoi(strings.TrimSpace(port))
		if err != nil {
			log.Printf("parameter %s is not valid number\n", port)
			return Config{}, ErrInvalidParameter
		}

		if ipPort < 0 || ipPort > math.MaxUint16 {
			log.Printf("parameter %s is not valid ip port\n", port)
			return Config{}, ErrInvalidParameter
		}

		ipPorts = append(ipPorts, uint16(ipPort))
	}

	return Config{localAddress: lip, remoteAddress: rip, ports: ipPorts}, nil
}
