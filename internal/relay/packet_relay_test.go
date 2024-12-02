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
	"errors"
	"io"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type connMock struct {
	readCnt  int
	writeCnt int
	read     func(int) (n int, err error)
	write    func(int) (n int, err error)
	closed   bool
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

		rel.realyPackets(in, out)

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

		go rel.relay(cm, rch, wch)

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

		go rel.realyPackets(in, out)

		rnd := rand.NewSource(time.Now().Unix())
		ms := rnd.Int63() % 3000

		time.Sleep(time.Duration(ms) * time.Millisecond)

		rel.wg.Wait()

		assert.LessOrEqual(t, 1, in.readCnt)
		assert.LessOrEqual(t, 1, in.writeCnt)

		assert.LessOrEqual(t, 1, out.readCnt)
		assert.LessOrEqual(t, 1, out.writeCnt)
	})
}

func TestChanToConnReleay(t *testing.T) {
	t.Parallel()

	t.Run("Success ch2conn crelay", func(t *testing.T) {
		t.Parallel()

		ch := make(chan []byte, 1)

		go chanToConnRelay(ch, &connMock{})

		close(ch)
	})

	t.Run("Success ch2conn relay", func(t *testing.T) {
		t.Parallel()

		ch := make(chan []byte)
		cm := &connMock{}

		go chanToConnRelay(ch, cm)

		ch <- make([]byte, 10)

		close(ch)

		assert.EqualValues(t, 1, cm.writeCnt)
	})

	t.Run("Success ch2conn write failed", func(t *testing.T) {
		t.Parallel()

		ch := make(chan []byte)
		cm := &connMock{
			write: func(_ int) (n int, err error) {
				return 0, errors.New("error")
			},
		}

		go chanToConnRelay(ch, cm)

		ch <- make([]byte, 10)

		close(ch)

		assert.EqualValues(t, 1, cm.writeCnt)
	})
}

func TestConnToChanReleay(t *testing.T) {
	t.Parallel()

	t.Run("Success read and close", func(t *testing.T) {
		t.Parallel()

		ch := make(chan []byte, 1)
		cm := &connMock{
			read: func(cnt int) (n int, err error) {
				if cnt > 1 {
					return 0, io.EOF
				}
				return 5, nil
			},
		}

		connToChanRelay(cm, ch)

		data, ok := <-ch

		assert.True(t, ok)
		assert.NotNil(t, data)

		assert.EqualValues(t, 2, cm.readCnt)
	})

	t.Run("Fail with not EOF", func(t *testing.T) {
		t.Parallel()

		ch := make(chan []byte, 1)
		cm := &connMock{
			read: func(_ int) (n int, err error) {
				return 0, errors.New("error")
			},
		}

		connToChanRelay(cm, ch)

		assert.EqualValues(t, 1, cm.readCnt)
	})
}

func TestMakeReleay(t *testing.T) {
	t.Parallel()

	prelay := newPacketRelay()

	assert.NotNil(t, prelay.wg)
}

// Mockup net.Conn
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
func (connMock) LocalAddr() net.Addr {
	return nil
}
func (connMock) RemoteAddr() net.Addr {
	return nil
}
func (connMock) SetDeadline(t time.Time) error {
	return nil
}
func (connMock) SetReadDeadline(t time.Time) error {
	return nil
}
func (connMock) SetWriteDeadline(t time.Time) error {
	return nil
}
