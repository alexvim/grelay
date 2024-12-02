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
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestListener(t *testing.T) {
	t.Parallel()

	t.Run("Success on listen and accept", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			_ = listenConn(ctx, "127.0.0.1:50500", func(_ context.Context, conn net.Conn) {
				defer conn.Close()
				cancel()
			})
		}()

		time.Sleep(time.Second)

		conn, err := net.Dial("tcp", "127.0.0.1:50500")

		assert.NoError(t, err)

		conn.Close()
	})

	t.Run("Many_conn_up_and_close", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		wg, barier, connCount := &sync.WaitGroup{}, &sync.WaitGroup{}, &sync.WaitGroup{}

		barier.Add(1)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			connCount.Add(1)

			go func() {
				defer wg.Done()
				barier.Wait()

				conn, err := net.Dial("tcp", "127.0.0.1:51110")
				assert.NoError(t, err)
				assert.NotNil(t, conn)

				conn.Read(make([]byte, 10))
				conn.Close()
			}()
		}

		go func() {
			err := listenConn(ctx, "127.0.0.1:51110", func(lctx context.Context, conn net.Conn) {
				connCount.Done()
				<-lctx.Done()
				conn.Close()
			})

			assert.NoError(t, err)
		}()

		time.Sleep(time.Second)

		barier.Done()

		connCount.Wait()

		cancel()

		wg.Wait()
	})

	t.Run("Fail to connect", func(t *testing.T) {
		t.Parallel()

		err := listenConn(context.Background(), "428.0.0.1:1012", func(_ context.Context, _ net.Conn) {})

		assert.Error(t, err)
	})
}
