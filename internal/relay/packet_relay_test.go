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
	"testing"

	"github.com/stretchr/testify/assert"
)

const mockRemoteAddr = "mock-remote-addr"

type rmMock struct {
	readCnt  int
	writeCnt int
}

func TestChanToConnReleay(t *testing.T) {
	t.Parallel()

	t.Run("Success_ch2conn_relay", func(t *testing.T) {
		t.Parallel()

		ch, cm := make(chan []byte, 1), &rmMock{}

		go chanToConnRelay(cm, ch, mockRemoteAddr)

		close(ch)
	})

	t.Run("Success_ch2conn_write_relay", func(t *testing.T) {
		t.Parallel()

		ch, cm := make(chan []byte), &rmMock{}

		go chanToConnRelay(cm, ch, mockRemoteAddr)

		ch <- make([]byte, 10)

		close(ch)

		assert.EqualValues(t, 1, cm.writeCnt)
	})

	t.Run("Success_ch2conn_write_failed", func(t *testing.T) {
		t.Parallel()

		ch, cm := make(chan []byte), &connMock{write: func(_ int) (n int, err error) { return 0, errors.New("error") }}

		go chanToConnRelay(cm, ch, mockRemoteAddr)

		ch <- make([]byte, 10)

		close(ch)

		assert.EqualValues(t, 1, cm.writeCnt)
	})
}

func TestConnToChanReleay(t *testing.T) {
	t.Parallel()

	t.Run("Success_read_and_close", func(t *testing.T) {
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

		connToChanRelay(cm, ch, mockRemoteAddr)

		data, ok := <-ch

		assert.True(t, ok)
		assert.NotNil(t, data)

		assert.EqualValues(t, 2, cm.readCnt)
	})

	t.Run("Fail_with_not_EOF", func(t *testing.T) {
		t.Parallel()

		ch, cm := make(chan []byte, 1), &connMock{read: func(_ int) (n int, err error) { return 0, errors.New("error") }}

		connToChanRelay(cm, ch, mockRemoteAddr)

		assert.EqualValues(t, 1, cm.readCnt)
	})
}

func TestMakeReleay(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, newPacketRelay().wg)
}

// Mockup io.ReaderWriteCloser
func (conn *rmMock) Read(b []byte) (n int, err error) {
	conn.readCnt++
	return 5, nil
}

func (conn *rmMock) Write(b []byte) (n int, err error) {
	conn.writeCnt++
	return 5, nil
}
