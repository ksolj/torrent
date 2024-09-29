package torrentfile

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"torrent/p2p"

	"github.com/jackpal/bencode-go"
)

const Port uint16 = 6881

type TorrentFile struct {
	TrackerURLs []string // Announce replacement (Tracker URLs)
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Files       []File
	Length      int
	Name        string
}

type File struct {
	Length int
	Path   string
}

type bencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Name        string `bencode:"name"`
	Length      int    `bencode:"length"`
	Files       []struct {
		Length int      `bencode:"length"` // The length of the file, in bytes.
		Path   []string `bencode:"path"`   // List of subdirectories, last is file name
	} `bencode:"files"`
}

type bencodeTorrent struct {
	Announce     string      `bencode:"announce"`
	AnnounceList [][]string  `bencode:"announce-list"`
	Info         bencodeInfo `bencode:"info"`
}

func (t *TorrentFile) DownloadToFile(path string) error {
	log.Printf("Number of files is %d", len(t.Files))
	for _, file := range t.Files {
		log.Print(file.Path)
	}
	log.Printf("Number of trackers is %d", len(t.TrackerURLs))
	log.Print("Current trackers")
	for _, st := range t.TrackerURLs {
		log.Print(st)
	}
	log.Printf("Announce URL: %v", t.TrackerURLs[0])

	var peerID [20]byte
	_, err := rand.Read(peerID[:])
	if err != nil {
		return err
	}

	peers, err := t.requestPeers(peerID, Port)
	if err != nil {
		return err
	}

	torrent := p2p.Torrent{
		Peers:       peers,
		PeerID:      peerID,
		InfoHash:    t.InfoHash,
		PieceHashes: t.PieceHashes,
		PieceLength: t.PieceLength,
		Length:      t.Length,
		Name:        t.Name,
	}
	buf, err := torrent.Download()
	if err != nil {
		return err
	}

	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outFile.Close()
	_, err = outFile.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

// a bit updated version
func Open(path string) (TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return TorrentFile{}, err
	}
	defer file.Close()

	bto := bencodeTorrent{}
	err = bencode.Unmarshal(file, &bto)
	if err != nil {
		return TorrentFile{}, err
	}
	return bto.toTorrentFile()
}

func (i *bencodeInfo) hash() ([20]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *i)
	if err != nil {
		return [20]byte{}, err
	}
	h := sha1.Sum(buf.Bytes())
	return h, nil
}

func (i *bencodeInfo) splitPieceHashes() ([][20]byte, error) {
	hashLen := 20 // Length of SHA-1 hash
	buf := []byte(i.Pieces)
	if len(buf)%hashLen != 0 {
		err := fmt.Errorf("Received malformed pieces of length %d", len(buf))
		return nil, err
	}
	numHashes := len(buf) / hashLen
	hashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
	}
	return hashes, nil
}

func (bto *bencodeTorrent) toTorrentFile() (TorrentFile, error) {
	infoHash, err := bto.Info.hash()
	if err != nil {
		return TorrentFile{}, err
	}
	pieceHashes, err := bto.Info.splitPieceHashes()
	if err != nil {
		return TorrentFile{}, err
	}
	var trackerURLs []string
	for _, list := range bto.AnnounceList {
		trackerURLs = append(trackerURLs, list...)
	}

	if len(trackerURLs) == 0 {
		trackerURLs = append(trackerURLs, bto.Announce)
	}

	t := TorrentFile{
		TrackerURLs: trackerURLs,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: bto.Info.PieceLength,
		Name:        bto.Info.Name,
	}

	if bto.Info.Length != 0 {
		t.Files = append(t.Files, File{
			Length: bto.Info.Length,
			Path:   bto.Info.Name,
		})
		t.Length = bto.Info.Length
	} else {
		for _, file := range bto.Info.Files {
			log.Print("test")
			listPath := append([]string{bto.Info.Name}, file.Path...)
			t.Files = append(t.Files, File{
				Length: file.Length,
				Path:   filepath.Join(listPath...),
			})
			t.Length += file.Length
		}
	}

	return t, nil
}
