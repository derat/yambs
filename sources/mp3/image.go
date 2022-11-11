// Copyright 2022 Daniel Erat.
// All rights reserved.

package mp3

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/derat/taglib-go/taglib"
	"github.com/derat/taglib-go/taglib/id3"
)

const (
	// Per "4.15. Attached picture" from https://id3.org/id3v2.3.0
	// and "4.14. Attached picture" from https://id3.org/id3v2.4.0-frames.
	imgFrameID = "APIC"

	imgBufLen = 128
)

// imgInfo describes an image embedded in an MP3 file.
type imgInfo struct {
	desc string // human-readable description, e.g. "front cover, 435422 bytes"
	path string // path to temp file containing image data
}

// getImages returns information about images stored in gen.
// Image data is written to temporary files.
func getImages(gen taglib.GenericTag) ([]imgInfo, error) {
	var infos []imgInfo
	switch tag := gen.(type) {
	case *id3.Id3v23Tag:
		for _, frame := range tag.Frames[imgFrameID] {
			info, err := readImageFrame(frame.Content)
			if err != nil {
				return nil, err
			}
			infos = append(infos, info)
		}
	case *id3.Id3v24Tag:
		for _, frame := range tag.Frames[imgFrameID] {
			info, err := readImageFrame(frame.Content)
			if err != nil {
				return nil, err
			}
			infos = append(infos, info)
		}
	default:
		return nil, errors.New("unsupported ID3 version")
	}
	return infos, nil
}

// readImageFrame reads an ID3v2.3 or v2.4 APIC frame's content.
func readImageFrame(data []byte) (imgInfo, error) {
	r := bytes.NewReader(data)
	var rerr error
	read := func(dst interface{}) {
		if rerr == nil {
			// "3. ID3v2 overview" from https://id3.org/id3v2.4.0-structure:
			// "The bitorder in ID3v2 is most significant bit first (MSB). The byteorder in
			// multibyte numbers is most significant byte first (e.g. $12345678 would be encoded $12
			// 34 56 78), also known as big endian and network byte order."
			rerr = binary.Read(r, binary.BigEndian, dst)
		}
	}

	// "4.14. Attached picture" from https://id3.org/id3v2.4.0-frames:
	//  Text encoding      $xx
	//  MIME type          <text string> $00
	//  Picture type       $xx
	//  Description        <text string according to encoding> $00 (00)
	//  Picture data       <binary data>
	var enc, picType byte
	var mimeType, desc string

	read(&enc)

	// This is a huge abuse of the taglib package. Shove the encoding byte and everything up to and
	// including the next 0x0 byte into a frame and pass it to taglib so we don't need to duplicate
	// the code for handling different encodings.
	readString := func(dst *string) {
		if rerr != nil {
			return
		}
		frame := id3.Id3v24Frame{Content: make([]byte, 0, imgBufLen)}
		frame.Content = append(frame.Content, enc)
		for {
			var ch byte
			if read(&ch); rerr != nil {
				return
			}
			frame.Content = append(frame.Content, ch)
			if ch == 0x0 {
				break
			}
		}
		if vals, err := id3.GetId3v24TextIdentificationFrame(&frame); err != nil {
			rerr = err
		} else if len(vals) != 1 {
			rerr = fmt.Errorf("got %d fields when reading string", len(vals))
		} else {
			*dst = vals[0]
		}
	}

	readString(&mimeType)
	read(&picType)
	readString(&desc)

	var img imgInfo
	if rerr != nil {
		return img, rerr
	}

	if img.desc = pictureTypes[picType]; len(img.desc) == 0 {
		img.desc = fmt.Sprintf("unknown (%#x)", picType)
	}
	img.desc += fmt.Sprintf(", %d bytes", r.Len())

	// Write the data to a temporary file.
	tf, err := ioutil.TempFile("", fmt.Sprintf("yambs-%s-img-*%s",
		time.Now().Format("20060102-150405"), getImageExt(mimeType)))
	if err != nil {
		return img, err
	}
	if _, err := io.Copy(tf, r); err != nil {
		os.Remove(tf.Name())
		return img, err
	}
	img.path = tf.Name()

	return img, nil
}

// getImageExt returns an appropriate file extension for a MIME type from an APIC frame.
func getImageExt(mimeType string) string {
	switch strings.ToLower(mimeType) {
	case "image/png", "png":
		return ".png"
	case "image/jpeg", "jpg":
		return ".jpg"
	default:
		return ".bin"
	}
}

// Based on "4.14. Attached picture" in https://id3.org/id3v2.4.0-frames.
var pictureTypes = map[byte]string{
	0x00: "other",
	0x01: "32x32 file icon",
	0x02: "other file icon",
	0x03: "front cover",
	0x04: "back cover",
	0x05: "leaflet page",
	0x06: "media",
	0x07: "lead artist/lead performer/soloist",
	0x08: "artist/performer",
	0x09: "conductor",
	0x0A: "band/orchestra",
	0x0B: "composer",
	0x0C: "lyricist/text writer",
	0x0D: "recording location",
	0x0E: "during recording",
	0x0F: "during performance",
	0x10: "movie/video screen capture",
	0x11: "a bright coloured fish", // okay, sure...
	0x12: "illustration",
	0x13: "band/artist logotype",
	0x14: "publisher/studio logotype",
}
