package main

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTelnetClient(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:")
		require.NoError(t, err)
		defer func() { require.NoError(t, l.Close()) }()

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()

			in := &bytes.Buffer{}
			out := &bytes.Buffer{}

			timeout, err := time.ParseDuration("10s")
			require.NoError(t, err)

			client := NewTelnetClient(l.Addr().String(), timeout, io.NopCloser(in), out)
			require.NoError(t, client.Connect())
			defer func() { require.NoError(t, client.Close()) }()

			in.WriteString("hello\n")
			err = client.Send()
			require.NoError(t, err)

			err = client.Receive()
			require.NoError(t, err)
			require.Equal(t, "world\n", out.String())
		}()

		go func() {
			defer wg.Done()

			conn, err := l.Accept()
			require.NoError(t, err)
			require.NotNil(t, conn)
			defer func() { require.NoError(t, conn.Close()) }()

			request := make([]byte, 1024)
			n, err := conn.Read(request)
			require.NoError(t, err)
			require.Equal(t, "hello\n", string(request)[:n])

			n, err = conn.Write([]byte("world\n"))
			require.NoError(t, err)
			require.NotEqual(t, 0, n)
		}()

		wg.Wait()
	})

	// Подключение к порту, который никто не слушает
	t.Run("connection refused", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:")
		require.NoError(t, err)
		addr := l.Addr().String()
		require.NoError(t, l.Close())

		client := NewTelnetClient(addr, time.Second, io.NopCloser(&bytes.Buffer{}), &bytes.Buffer{})
		require.Error(t, client.Connect())

		require.NoError(t, client.Close())
	})

	// Некорректный адрес сети. Нет порта
	t.Run("invalid address", func(t *testing.T) {
		client := NewTelnetClient("localhost", time.Second, io.NopCloser(&bytes.Buffer{}), &bytes.Buffer{})
		err := client.Connect()
		require.Error(t, err)
		var opErr *net.OpError
		require.ErrorAs(t, err, &opErr)
	})

	t.Run("close without connect", func(t *testing.T) {
		client := NewTelnetClient("127.0.0.1:0", time.Second, io.NopCloser(&bytes.Buffer{}), &bytes.Buffer{})
		require.NoError(t, client.Close())
	})

	// Сервер закрыл соединение
	t.Run("closed by peer", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:")
		require.NoError(t, err)
		defer l.Close()

		go func() {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}()

		in := &bytes.Buffer{}
		client := NewTelnetClient(l.Addr().String(), time.Second, io.NopCloser(in), &bytes.Buffer{})
		require.NoError(t, client.Connect())
		defer client.Close()

		require.NoError(t, client.Receive())

		for i := 0; i < 10; i++ {
			in.WriteString("ping\n")
			if err = client.Send(); err != nil {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		require.Error(t, err)
	})
}
