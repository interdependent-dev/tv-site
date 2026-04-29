package main

import (
	"bufio"
	"context"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type LogLine struct {
	Source string    `json:"source"`
	Time   time.Time `json:"time"`
	Level  string    `json:"level"`
	Text   string    `json:"text"`
}

type LogHub struct {
	mu      sync.RWMutex
	files   map[string]string
	buffers map[string][]LogLine
	subs    map[chan LogLine]struct{}
}

const historyMax = 500

func NewLogHub() *LogHub {
	return &LogHub{
		files:   map[string]string{},
		buffers: map[string][]LogLine{},
		subs:    map[chan LogLine]struct{}{},
	}
}

func (h *LogHub) Add(name, path string) { h.files[name] = path }

func (h *LogHub) Start(ctx context.Context) {
	for name, path := range h.files {
		go h.tailFile(ctx, name, path)
	}
}

func (h *LogHub) Subscribe() (chan LogLine, func()) {
	ch := make(chan LogLine, 200)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
		close(ch)
	}
}

func (h *LogHub) Publish(line LogLine) {
	h.mu.Lock()
	buf := h.buffers[line.Source]
	buf = append(buf, line)
	if len(buf) > historyMax {
		buf = buf[len(buf)-historyMax:]
	}
	h.buffers[line.Source] = buf
	for ch := range h.subs {
		select {
		case ch <- line:
		default:
		}
	}
	h.mu.Unlock()
}

func (h *LogHub) History(source string) []LogLine {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]LogLine, len(h.buffers[source]))
	copy(out, h.buffers[source])
	return out
}

func (h *LogHub) AllHistory() []LogLine {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var all []LogLine
	for _, buf := range h.buffers {
		all = append(all, buf...)
	}
	return all
}

var levelPattern = regexp.MustCompile(`\b(FTL|ERR|ERROR|WARN?|WRN|INF|INFO|DEB|DBG|DEBUG|TRC|TRACE)\b`)

func detectLevel(s string) string {
	m := levelPattern.FindString(s)
	if m == "" {
		return "INF"
	}
	switch strings.ToUpper(m) {
	case "FTL", "ERR", "ERROR":
		return "ERR"
	case "WAR", "WARN", "WRN":
		return "WRN"
	case "DEB", "DBG", "DEBUG", "TRC", "TRACE":
		return "DBG"
	}
	return "INF"
}

func (h *LogHub) tailFile(ctx context.Context, name, path string) {
	var f *os.File
	var reader *bufio.Reader
	var lastSize int64

	reopen := func() bool {
		if f != nil {
			f.Close()
		}
		var err error
		f, err = os.Open(path)
		if err != nil {
			return false
		}
		// On first open, seek back ~64KB so recent history populates the
		// ring buffer + health monitor. (Pure tail-from-end means the
		// dashboard sees nothing until a new line arrives.) Skip the
		// first partial line so we never publish a half-line.
		const tailBytes int64 = 64 * 1024
		info, _ := f.Stat()
		if info != nil {
			start := info.Size() - tailBytes
			if start < 0 {
				start = 0
			}
			f.Seek(start, io.SeekStart)
			if start > 0 {
				// discard up to the next newline so we don't emit a partial line
				br := bufio.NewReader(f)
				_, _ = br.ReadString('\n')
				pos, _ := f.Seek(0, io.SeekCurrent)
				_ = pos
				reader = br
			} else {
				reader = bufio.NewReader(f)
			}
			lastSize, _ = f.Seek(0, io.SeekCurrent)
		} else {
			reader = bufio.NewReader(f)
		}
		return true
	}

	tick := time.NewTicker(400 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			if f != nil {
				f.Close()
			}
			return
		case <-tick.C:
		}

		if f == nil {
			if !reopen() {
				continue
			}
		}

		info, err := os.Stat(path)
		if err != nil {
			f.Close()
			f = nil
			continue
		}
		if info.Size() < lastSize {
			// truncated / rotated — reopen from start
			if f != nil {
				f.Close()
			}
			f = nil
			lastSize = 0
			continue
		}

		for {
			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				lastSize, _ = f.Seek(0, io.SeekCurrent)
				text := strings.TrimRight(line, "\r\n")
				if text == "" {
					continue
				}
				h.Publish(LogLine{
					Source: name,
					Time:   time.Now(),
					Level:  detectLevel(text),
					Text:   text,
				})
			}
			if err != nil {
				break
			}
		}
	}
}
