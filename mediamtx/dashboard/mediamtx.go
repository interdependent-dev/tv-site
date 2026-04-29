package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Path struct {
	Name          string   `json:"name"`
	ConfName      string   `json:"confName"`
	Source        any      `json:"source"`
	Ready         bool     `json:"ready"`
	ReadyTime     string   `json:"readyTime"`
	Tracks        []string `json:"tracks"`
	BytesReceived int64    `json:"bytesReceived"`
	BytesSent     int64    `json:"bytesSent"`
	Readers       []any    `json:"readers"`
}

type PathList struct {
	PageCount int    `json:"pageCount"`
	ItemCount int    `json:"itemCount"`
	Items     []Path `json:"items"`
}

type HLSMuxer struct {
	Path         string `json:"path"`
	Created      string `json:"created"`
	LastRequest  string `json:"lastRequest"`
	BytesSent    int64  `json:"bytesSent"`
}

type HLSMuxerList struct {
	ItemCount int        `json:"itemCount"`
	Items     []HLSMuxer `json:"items"`
}

type SRTConn struct {
	ID            string `json:"id"`
	Created       string `json:"created"`
	RemoteAddr    string `json:"remoteAddr"`
	State         string `json:"state"`
	Path          string `json:"path"`
	BytesReceived int64  `json:"bytesReceived"`
	BytesSent     int64  `json:"bytesSent"`
}

type SRTConnList struct {
	ItemCount int       `json:"itemCount"`
	Items     []SRTConn `json:"items"`
}

type WebRTCSession struct {
	ID                 string `json:"id"`
	Created            string `json:"created"`
	RemoteAddr         string `json:"remoteAddr"`
	Path               string `json:"path"`
	State              string `json:"state"`
	BytesReceived      int64  `json:"bytesReceived"`
	BytesSent          int64  `json:"bytesSent"`
	RTCPeerConnState   string `json:"peerConnectionEstablished"`
}

type WebRTCSessionList struct {
	ItemCount int             `json:"itemCount"`
	Items     []WebRTCSession `json:"items"`
}

var mtxClient = &http.Client{Timeout: 4 * time.Second}

func fetchMTX[T any](ctx context.Context, base, suffix string) (T, error) {
	var out T
	req, err := http.NewRequestWithContext(ctx, "GET", base+"/v3/"+suffix, nil)
	if err != nil {
		return out, err
	}
	resp, err := mtxClient.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return out, fmt.Errorf("mediamtx %s → %d", suffix, resp.StatusCode)
	}
	err = json.NewDecoder(resp.Body).Decode(&out)
	return out, err
}

func FetchPaths(ctx context.Context, base string) (PathList, error) {
	return fetchMTX[PathList](ctx, base, "paths/list")
}

func FetchHLSMuxers(ctx context.Context, base string) (HLSMuxerList, error) {
	return fetchMTX[HLSMuxerList](ctx, base, "hlsmuxers/list")
}

func FetchSRTConns(ctx context.Context, base string) (SRTConnList, error) {
	return fetchMTX[SRTConnList](ctx, base, "srtconns/list")
}

func FetchWebRTCSessions(ctx context.Context, base string) (WebRTCSessionList, error) {
	return fetchMTX[WebRTCSessionList](ctx, base, "webrtcsessions/list")
}

func countPublishers(paths PathList) int {
	n := 0
	for _, p := range paths.Items {
		if p.Ready {
			n++
		}
	}
	return n
}

func countReaders(paths PathList) int {
	n := 0
	for _, p := range paths.Items {
		n += len(p.Readers)
	}
	return n
}
