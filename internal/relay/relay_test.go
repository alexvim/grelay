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
	"math/rand"
	"net"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockConfig struct{}

type connMock struct {
	readCnt  int
	writeCnt int
	read     func(int) (n int, err error)
	write    func(int) (n int, err error)
	closed   bool
}

func TestRelay(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	go Run(ctx, mockConfig{})

	time.Sleep(time.Second)

	cancel()
}

func TestReleay(t *testing.T) {
	t.Parallel()

	t.Run("Success_packets_relay", func(t *testing.T) {
		t.Parallel()

		mockFunc := func(cnt int) (n int, err error) {
			if cnt > 1000 {
				return 0, io.EOF
			}
			return 5, nil
		}

		in := &connMock{read: mockFunc, write: mockFunc}
		out := &connMock{read: mockFunc, write: mockFunc}

		rel := newPacketRelay()

		rel.realyPackets(in, "in-remote-addr", out, "out-remote-addr")

		assert.LessOrEqual(t, 1, in.readCnt)
		assert.LessOrEqual(t, 1, in.writeCnt)

		assert.LessOrEqual(t, 1, out.readCnt)
		assert.LessOrEqual(t, 1, out.writeCnt)
	})

	t.Run("Success_relay", func(t *testing.T) {
		t.Parallel()

		mockFunc := func(cnt int) (n int, err error) {
			if cnt > 1 {
				return 0, io.EOF
			}
			return 5, nil
		}

		cm := &connMock{read: mockFunc, write: mockFunc}

		wch, rch := make(chan []byte, 1), make(chan []byte, 1)

		rel := newPacketRelay()

		rch <- make([]byte, 1)

		go rel.relay(cm, "remote-addr", rch, wch)

		<-wch

		close(rch)

		rel.wg.Wait()

		assert.LessOrEqual(t, 1, cm.readCnt)
		assert.LessOrEqual(t, 1, cm.writeCnt)
	})

	t.Run("Terminate_relay", func(t *testing.T) {
		t.Parallel()

		mockFunc := func(cnt int) (n int, err error) {
			if cnt > 1000 {
				return 0, io.EOF
			}
			return 5, nil
		}

		in := &connMock{read: mockFunc, write: mockFunc}
		out := &connMock{read: mockFunc, write: mockFunc}

		rel := newPacketRelay()

		go rel.realyPackets(in, "in-remote-addr", out, "out-remote-addr")

		rnd := rand.NewSource(time.Now().Unix())
		ms := rnd.Int63() % 3000

		time.Sleep(time.Duration(ms) * time.Millisecond)

		rel.wg.Wait()

		assert.LessOrEqual(t, 1, in.readCnt)
		assert.LessOrEqual(t, 1, in.writeCnt)

		assert.LessOrEqual(t, 1, out.readCnt)
		assert.LessOrEqual(t, 1, out.writeCnt)
	})

	t.Run("Success_run_relay", func(t *testing.T) {
		const localAddress = "127.0.0.1:50110"
		const remoteAddress = "127.0.0.1:50011"

		ctx, cancel := context.WithCancel(context.Background())

		rel := newPacketRelay()

		wg := &sync.WaitGroup{}

		wg.Add(1)
		go func() {
			rel.runRelay(ctx, localAddress, remoteAddress)
			wg.Done()
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			listen, err := net.Listen("tcp", remoteAddress)
			if err != nil {
				assert.FailNow(t, "failed to open listener")
			}

			conn, err := listen.Accept()
			if err != nil {
				assert.FailNow(t, "failed to accept conn")
			}

			conn.Close()
		}()

		time.Sleep(time.Second)

		conn, err := net.Dial("tcp", localAddress)
		if err != nil {
			assert.Fail(t, "error on test dial")

			defer cancel()

			return
		}

		defer conn.Close()

		cancel()

		wg.Wait()
	})

	t.Run("Failed_to_conn_on_run_relay", func(t *testing.T) {
		const localAddress = "127.0.0.1:50112"

		ctx, cancel := context.WithCancel(context.Background())

		rel := newPacketRelay()

		wg := &sync.WaitGroup{}

		wg.Add(1)
		go func() {
			rel.runRelay(ctx, localAddress, "127.0.0.1:50012")
			wg.Done()
		}()

		time.Sleep(time.Second)

		conn, err := net.Dial("tcp", localAddress)
		if err != nil {
			defer cancel()

			assert.Fail(t, "error on test dial")
			return
		}

		defer conn.Close()

		cancel()

		wg.Wait()
	})
}

func TestUtils(t *testing.T) {
	t.Parallel()

	t.Run("Success on listen and accept", func(t *testing.T) {
		addr := makeAddr(netip.MustParseAddr("127.0.0.1"), 3030)

		assert.EqualValues(t, "127.0.0.1:3030", addr)
	})
}

// Mockup io.ReaderWriteCloser
func (conn *connMock) Read(b []byte) (n int, err error) {
	if conn.closed {
		return 0, io.EOF
	}

	conn.readCnt++
	if conn.read != nil {
		return conn.read(conn.readCnt)
	}

	return 5, nil
}

func (conn *connMock) Write(b []byte) (n int, err error) {
	if conn.closed {
		return 0, io.EOF
	}

	conn.writeCnt++
	if conn.write != nil {
		return conn.write(conn.writeCnt)
	}

	return 5, nil
}

func (conn *connMock) Close() error {
	conn.closed = true
	return nil
}

func (mockConfig) Local() netip.Addr {
	return netip.MustParseAddr("127.0.0.1")
}

func (mockConfig) Remote() netip.Addr {
	return netip.MustParseAddr("127.0.0.1")
}

func (mockConfig) Ports() []uint16 {
	return []uint16{30000}
}
