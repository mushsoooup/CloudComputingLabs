package pool

import (
	"io"
	"main/logger"
	"net"
	"sync"
	"time"
)

const maxIdle = time.Second * 2

type ConnHandler func(net.Conn) error

// Basic goroutine pool struct
type Pool struct {
	workerFunc ConnHandler
	lock       sync.Mutex

	cocurrency     int
	ready          []*workerChan
	workersCnt     int
	workerChanPool sync.Pool
}
type workerChan struct {
	lastUsed time.Time
	ch       chan net.Conn
}

func (p *Pool) get() *workerChan {
	var ch *workerChan

	new := false

	p.lock.Lock()
	ready := p.ready
	n := len(ready) - 1
	if n < 0 {
		if p.workersCnt < p.cocurrency {
			new = true
			p.workersCnt++
		}
	} else {
		ch = ready[n]
		ready[n] = nil
		p.ready = ready[:n]
	}
	p.lock.Unlock()

	if ch == nil && new {
		vch := p.workerChanPool.Get()
		ch = vch.(*workerChan)
		go func() {
			p.worker(ch)
			p.workerChanPool.Put(vch)
		}()
	}

	return ch
}

func (p *Pool) worker(ch *workerChan) {
	var c net.Conn
	var err error
	for c = range ch.ch {
		if c == nil {
			break
		}
		if err = p.workerFunc(c); err != nil && err != io.EOF {
			logger.Debug("error serving %q<->%q: %v", c.LocalAddr(), c.RemoteAddr(), err)
		}
		_ = c.Close()
		ch.lastUsed = time.Now()
		p.lock.Lock()
		p.ready = append(p.ready, ch)
		p.lock.Unlock()
	}
	p.lock.Lock()
	p.workersCnt--
	p.lock.Unlock()
}

func (p *Pool) Start(handler ConnHandler, cocurrency int) {
	p.workerFunc = handler
	p.cocurrency = cocurrency
	p.workerChanPool.New = func() any {
		return &workerChan{
			ch: make(chan net.Conn, 1),
		}
	}
	// Periodically clean unused worker
	go func() {
		var scratch []*workerChan
		for {
			p.clean(&scratch)
			time.Sleep(maxIdle)
		}
	}()
}

func (p *Pool) Serve(c net.Conn) bool {
	ch := p.get()
	if ch == nil {
		return false
	}
	ch.ch <- c
	return true
}

func (p *Pool) clean(scratch *[]*workerChan) {
	criticalTime := time.Now().Add(maxIdle)

	p.lock.Lock()
	ready := p.ready
	n := len(ready)

	// Binary search
	l, r := 0, n-1
	for l <= r {
		mid := (l + r) / 2
		if criticalTime.After(p.ready[mid].lastUsed) {
			l = mid + 1
		} else {
			r = mid - 1
		}
	}
	i := r
	if i == -1 {
		p.lock.Unlock()
		return
	}
	*scratch = append((*scratch)[:0], ready[:i+1]...)
	m := copy(ready, ready[i+1:])
	for i = m; i < n; i++ {
		ready[i] = nil
	}
	p.ready = ready[:m]
	p.lock.Unlock()

	tmp := *scratch
	for i := range tmp {
		tmp[i].ch <- nil
		tmp[i] = nil
	}
}
