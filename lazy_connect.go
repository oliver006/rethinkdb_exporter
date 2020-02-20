package main

import (
	"crypto/tls"
	"golang.org/x/net/context"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
	"log"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	notInited = 0
	inited    = 1
)

type lazySession struct {
	*r.Session
	inited int32

	opts r.ConnectOpts
	m    sync.Mutex
}

func (l *lazySession) Close() error {
	if atomic.LoadInt32(&l.inited) == inited {
		return l.Session.Close()
	}
	return nil
}

func (l *lazySession) IsConnected() bool {
	if atomic.LoadInt32(&l.inited) == notInited {
		err := l.connect()
		if err != nil {
			log.Printf("failed to connect to rethinkdb: %v", err)
			return false
		}
	}

	is := l.Session.IsConnected()
	if !is {
		err := l.Session.Reconnect()
		if err != nil {
			return false
		}
		is = l.Session.IsConnected()
	}
	return is
}

func (l *lazySession) Query(ctx context.Context, q r.Query) (*r.Cursor, error) {
	if atomic.LoadInt32(&l.inited) == notInited {
		err := l.connect()
		if err != nil {
			return nil, err
		}
	}

	cur, err := l.Session.Query(ctx, q)
	if err == r.ErrConnectionClosed {
		err = l.Session.Reconnect()
		if err != nil {
			return nil, err
		}
		cur, err = l.Session.Query(ctx, q)
	}
	return cur, err
}

func (l *lazySession) Exec(ctx context.Context, q r.Query) error {
	if atomic.LoadInt32(&l.inited) == notInited {
		err := l.connect()
		if err != nil {
			return err
		}
	}

	err := l.Session.Exec(ctx, q)
	if err == r.ErrConnectionClosed {
		err = l.Session.Reconnect()
		if err != nil {
			return err
		}
		err = l.Session.Exec(ctx, q)
	}
	return err
}

func (l *lazySession) connect() error {
	l.m.Lock()
	defer l.m.Unlock()

	var err error
	if atomic.LoadInt32(&l.inited) == notInited {
		l.Session, err = r.Connect(l.opts)
		if err != nil {
			// to connect at next attempt
			l.Session = nil
		}
		atomic.StoreInt32(&l.inited, inited)
	}
	return err
}

func connectRethinkdb(addr, auth, user, pass string, tlsConfig *tls.Config) *lazySession {
	return &lazySession{
		inited: notInited,
		opts: r.ConnectOpts{
			Addresses: strings.Split(addr, ","),
			Database:  "rethinkdb",
			AuthKey:   auth,
			Username:  user,
			Password:  pass,
			TLSConfig: tlsConfig,
			MaxOpen:   20,
		},
	}
}
