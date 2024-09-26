package torrentfile

// TODO: update this package according to og repo

import (
	"io"

	"github.com/jackpal/bencode-go"
)

type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

type bencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

func Open(r io.Reader) (*bencodeTorrent, error) {
	torrent := bencodeTorrent{}
	err := bencode.Unmarshal(r, &torrent)
	if err != nil {
		return nil, err
	}
	return &torrent, nil
}

func (bto bencodeTorrent) toTorrentFile() (TorrentFile, error) {
	// … TODO
}
