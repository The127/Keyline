package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"testing"

	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/logging"

	"github.com/The127/ioc"
	"github.com/stretchr/testify/require"
)

func freePort(t *testing.T) int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())

	return port
}

func serveOnFreePort(t *testing.T, dp *ioc.DependencyProvider) int {
	for range 5 {
		port := freePort(t)
		shutdown, err := Serve(dp, config.ServerConfig{Host: "127.0.0.1", Port: port})
		if errors.Is(err, syscall.EADDRINUSE) {
			continue
		}

		require.NoError(t, err)
		t.Cleanup(func() { _ = shutdown(context.Background()) })

		return port
	}

	t.Fatal("no free port found")
	return 0
}

func TestServeAcceptsConnectionsAsSoonAsItReturns(t *testing.T) {
	// arrange
	logging.Init()
	dp := ioc.NewDependencyCollection().BuildProvider()
	t.Cleanup(func() { require.NoError(t, dp.Close()) })

	// act
	port := serveOnFreePort(t, dp)

	// assert
	connection, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	require.NoError(t, connection.Close())
}

func TestServeRefusesAPortThatIsTaken(t *testing.T) {
	// arrange
	logging.Init()
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = taken.Close() })

	dp := ioc.NewDependencyCollection().BuildProvider()
	t.Cleanup(func() { require.NoError(t, dp.Close()) })

	// act
	_, err = Serve(dp, config.ServerConfig{Host: "127.0.0.1", Port: taken.Addr().(*net.TCPAddr).Port})

	// assert
	require.ErrorIs(t, err, syscall.EADDRINUSE)
}
