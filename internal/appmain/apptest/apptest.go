// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package apptest allows testing of binded services within memory.
package apptest

import (
	"net"
	"testing"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/appmain"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
)

func TestApp(t *testing.T, cfg config.View, listeners []net.Listener, binds ...appmain.Bind) {
	ls, err := newListenerStorage(listeners)
	if err != nil {
		t.Fatal(err)
	}
	getCfg := func() (config.View, error) {
		return cfg, nil
	}
	bindAll := func(p *appmain.Params, b *appmain.Bindings) error {
		for _, bind := range binds {
			err := bind(p, b)
			if err != nil {
				return err
			}
		}
		return nil
	}

	app, err := appmain.StartApplication("test", bindAll, getCfg, ls.listen)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		err := app.Stop()
		if err != nil {
			t.Fatal(err)
		}
	})
}

func newFullAddr(network, address string) (fullAddr, error) {
	a := fullAddr{
		network: network,
	}
	var err error
	a.host, a.port, err = net.SplitHostPort(address)
	if err != nil {
		return fullAddr{}, err
	}
	// Usually listeners are started with an "unspecified" ip address, which has
	// several equivilent forms: ":80", "0.0.0.0:80", "[::]:80".  Even if the
	// callers use the same form, the listeners may return a different form when
	// asked for its address.  So detect and revert to the simpler form.
	if net.ParseIP(a.host).IsUnspecified() {
		a.host = ""
	}
	return a, nil
}

type fullAddr struct {
	network string
	host    string
	port    string
}

type listenerStorage struct {
	l map[fullAddr]net.Listener
}

func newListenerStorage(listeners []net.Listener) (*listenerStorage, error) {
	ls := &listenerStorage{
		l: make(map[fullAddr]net.Listener),
	}
	for _, l := range listeners {
		a, err := newFullAddr(l.Addr().Network(), l.Addr().String())
		if err != nil {
			return nil, err
		}
		ls.l[a] = l
	}
	return ls, nil
}

func (ls *listenerStorage) listen(network, address string) (net.Listener, error) {
	a, err := newFullAddr(network, address)
	if err != nil {
		return nil, err
	}

	l, ok := ls.l[a]
	if ok {
		delete(ls.l, a)
		return l, nil
	}
	return nil, errors.Errorf("Listener for \"%s\" was not passed to TestApp or was already used", address)
}

func GRPCClient(t *testing.T, cfg config.View, service string) *grpc.ClientConn {
	conn, err := rpc.GRPCClientFromConfig(cfg, service)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		err := conn.Close()
		if err != nil {
			t.Fatal(err)
		}
	})
	return conn
}
