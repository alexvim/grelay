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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeConfig(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lip    string
		rip    string
		ports  string
		pports []uint16
		ok     bool
	}{
		"correct": {
			lip:    "129.23.22.123",
			rip:    "129.23.22.123",
			ports:  "443,23, 43, 432, 23423",
			pports: []uint16{443, 23, 43, 432, 23423},
			ok:     true,
		},
		"lip with port": {
			lip:    "129.23.22.123:3030",
			rip:    "129.23.22.123",
			ports:  "443,23, 43, 432, 23423",
			pports: []uint16{443, 23, 43, 432, 23423},
			ok:     false,
		},
		"rip with port": {
			lip:    "129.23.22.123",
			rip:    "129.23.22.123:3030",
			ports:  "443,23, 43, 432, 23423",
			pports: []uint16{443, 23, 43, 432, 23423},
			ok:     false,
		},
		"lip failed": {
			lip:    "129.23.22.323",
			rip:    "129.23.22.123",
			ports:  "443,23, 43, 432, 23423",
			pports: []uint16{443, 23, 43, 432, 23423},
			ok:     false,
		},
		"rip failed": {
			lip:    "129.23.22.123",
			rip:    "129.23.22.323",
			ports:  "443,23, 43, 432, 23423",
			pports: []uint16{443, 23, 43, 432, 23423},
			ok:     false,
		},
		"port format failed": {
			lip:    "129.23.22.123",
			rip:    "129.23.22.123",
			ports:  "443,23, 43, 432, s23423",
			pports: nil,
			ok:     false,
		},
		"port format failed by out of range": {
			lip:    "129.23.22.123",
			rip:    "129.23.22.123",
			ports:  "443,23, 43, 432, 232423",
			pports: []uint16{443, 23, 43, 432, 23423},
			ok:     false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg, err := makeConfig(test.lip, test.rip, test.ports)

			assert.EqualValues(t, test.ok, (err == nil), "input", test.lip, test.rip, test.ports)

			if err != nil {
				return
			}

			assert.EqualValues(t, test.lip, cfg.Local().String())
			assert.EqualValues(t, test.rip, cfg.Remote().String())
			assert.EqualValues(t, test.pports, cfg.Ports())
		})
	}
}

func TestNewConfig(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args []string
		ok   bool
	}{
		"correct": {
			args: []string{"-l", "129.23.22.123", "-r", "129.23.22.123", "-p", "443,23, 43, 432, 23423"},
			ok:   true,
		},
		"help": {
			args: []string{"-h"},
			ok:   false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := NewConfigFromCmdLineArgs(test.args)

			assert.Condition(t, func() bool {
				return (err == nil) == test.ok
			})
		})
	}
}

func TestPrint(t *testing.T) {
	t.Parallel()

	args := []string{"-l", "129.23.22.123", "-r", "129.23.22.123", "-p", "443,23, 43, 432, 23423"}

	cfg, err := NewConfigFromCmdLineArgs(args)

	assert.NoError(t, err)

	assert.NotEmpty(t, cfg.String())
}
