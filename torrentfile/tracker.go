package torrentfile

import (
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"torrent/peers"

	"github.com/jackpal/bencode-go"
)

type bencodeTrackerResp struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

func (t *TorrentFile) buildTrackerURLs(peerID [20]byte, port uint16) ([]string, error) {
	var TrackerURLs []string
	for _, v := range t.TrackerURLs {
		base, err := url.Parse(v)
		if err != nil {
			return TrackerURLs, err
		}
		params := url.Values{
			"info_hash":  []string{string(t.InfoHash[:])},
			"peer_id":    []string{string(peerID[:])},
			"port":       []string{strconv.Itoa(int(Port))},
			"uploaded":   []string{"0"},
			"downloaded": []string{"0"},
			"compact":    []string{"1"},
			"left":       []string{strconv.Itoa(t.Length)},
		}
		base.RawQuery = params.Encode()
		TrackerURLs = append(TrackerURLs, base.String())
		log.Print(base.String())
	}

	return TrackerURLs, nil
}

// This implementation of announce-list is kind of not right due to how it works
//
// It announces to all available trackers. This behavior is contrary to BEP12 actually
func (t *TorrentFile) requestPeers(peerID [20]byte, port uint16) ([]peers.Peer, error) {
	urls, err := t.buildTrackerURLs(peerID, port)
	if err != nil {
		return nil, err
	}

	var peersList []peers.Peer

	for _, url := range urls {
		c := &http.Client{Timeout: 15 * time.Second}
		resp, err := c.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		trackerResp := bencodeTrackerResp{}
		err = bencode.Unmarshal(resp.Body, &trackerResp)
		if err != nil {
			return nil, err
		}

		tempPeers, err := peers.Unmarshal([]byte(trackerResp.Peers))
		if err != nil {
			return nil, err
		}
		peersList = append(peersList, tempPeers...)
	}

	return peersList, nil
}
