// package url2svg ...
//
// forked from [github.com/aaronarduino/goqrsvg] [boomuler/barcode]
// This is an package internal minial code size | api stability fork!
// Please use always the original!
//
// [github.com/aaronarduino/goqrsvg] - Copyright (c) 2017 Aaron Alexander MIT License
// [github.com/boomuler/barcode] - Copyright (c) 2014 Florian Sundermann MIT License
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package url2svg

// import
import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

//
// EXTERNAL INTERFACE
//

// GetSVG returns an svg qr code from net/url
func GetSVG(u *url.URL) string {
	return GetStringSVG(u.String())
}

// GetStringSVG returns an svg qr code from string
func GetStringSVG(in string) string {
	if len(in) > 3500 {
		return ""
	}
	var buf bytes.Buffer
	s := New(&buf) // svg
	qrCode, err := Encode(in, M, Auto)
	if err != nil {
		return ""
	}
	qs := NewQrSVG(qrCode, 5)
	qs.StartQrSVG(s)
	qs.WriteQrSVG(s)
	s.End()
	out, err := io.ReadAll(&buf)
	if err != nil {
		return ""
	}
	return string(out)
}

//
// INTERNAL BACKEND
//

// QrSVG ...
type QrSVG struct {
	qr        Barcode
	qrWidth   int
	blockSize int
	startingX int
	startingY int
}

// NewQrSVG ...
func NewQrSVG(qr Barcode, blockSize int) QrSVG {
	return QrSVG{
		qr:        qr,
		qrWidth:   qr.Bounds().Max.X,
		blockSize: blockSize,
		startingX: 0,
		startingY: 0,
	}
}

// WriteQrSVG ...
func (qs *QrSVG) WriteQrSVG(s *SVG) error {
	if qs.qr.Metadata().CodeKind == "QR Code" {
		currY := qs.startingY

		for x := 0; x < qs.qrWidth; x++ {
			currX := qs.startingX
			for y := 0; y < qs.qrWidth; y++ {
				if qs.qr.At(x, y) == color.Black {
					s.Rect(currX, currY, qs.blockSize, qs.blockSize, "fill:black;stroke:none")
				} else if qs.qr.At(x, y) == color.White {
					s.Rect(currX, currY, qs.blockSize, qs.blockSize, "fill:white;stroke:none")
				}
				currX += qs.blockSize
			}
			currY += qs.blockSize
		}
		return nil
	}
	return errors.New("can not write to SVG: Not a QR code")
}

// StartQrSVG ...
func (qs *QrSVG) StartQrSVG(s *SVG) {
	width := (qs.qrWidth * qs.blockSize) + (qs.blockSize * 8)
	qs.setStartPoint(0, 0)
	s.Start(width, width)
}

func (qs *QrSVG) setStartPoint(x, y int) {
	qs.startingX = x + (qs.blockSize * 4)
	qs.startingY = y + (qs.blockSize * 4)
}

const charSet string = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"

func stringToAlphaIdx(content string) <-chan int {
	result := make(chan int)
	go func() {
		for _, r := range content {
			idx := strings.IndexRune(charSet, r)
			result <- idx
			if idx < 0 {
				break
			}
		}
		close(result)
	}()

	return result
}

func encodeAlphaNumeric(content string, ecl ErrorCorrectionLevel) (*BitList, *versionInfo, error) {
	contentLenIsOdd := len(content)%2 == 1
	contentBitCount := (len(content) / 2) * 11
	if contentLenIsOdd {
		contentBitCount += 6
	}
	vi := findSmallestVersionInfo(ecl, alphaNumericMode, contentBitCount)
	if vi == nil {
		return nil, nil, errors.New("To much data to encode")
	}

	res := new(BitList)
	res.AddBits(int(alphaNumericMode), 4)
	res.AddBits(len(content), vi.charCountBits(alphaNumericMode))

	encoder := stringToAlphaIdx(content)

	for idx := 0; idx < len(content)/2; idx++ {
		c1 := <-encoder
		c2 := <-encoder
		if c1 < 0 || c2 < 0 {
			return nil, nil, fmt.Errorf("\"%s\" can not be encoded as %s", content, AlphaNumeric)
		}
		res.AddBits(c1*45+c2, 11)
	}
	if contentLenIsOdd {
		c := <-encoder
		if c < 0 {
			return nil, nil, fmt.Errorf("\"%s\" can not be encoded as %s", content, AlphaNumeric)
		}
		res.AddBits(c, 6)
	}

	addPaddingAndTerminator(res, vi)

	return res, vi, nil
}

func encodeAuto(content string, ecl ErrorCorrectionLevel) (*BitList, *versionInfo, error) {
	bits, vi, _ := Numeric.getEncoder()(content, ecl)
	if bits != nil && vi != nil {
		return bits, vi, nil
	}
	bits, vi, _ = AlphaNumeric.getEncoder()(content, ecl)
	if bits != nil && vi != nil {
		return bits, vi, nil
	}
	bits, vi, _ = Unicode.getEncoder()(content, ecl)
	if bits != nil && vi != nil {
		return bits, vi, nil
	}
	return nil, nil, fmt.Errorf("No encoding found to encode \"%s\"", content)
}

type block struct {
	data []byte
	ecc  []byte
}
type blockList []*block

func splitToBlocks(data <-chan byte, vi *versionInfo) blockList {
	result := make(blockList, vi.NumberOfBlocksInGroup1+vi.NumberOfBlocksInGroup2)

	for b := 0; b < int(vi.NumberOfBlocksInGroup1); b++ {
		blk := new(block)
		blk.data = make([]byte, vi.DataCodeWordsPerBlockInGroup1)
		for cw := 0; cw < int(vi.DataCodeWordsPerBlockInGroup1); cw++ {
			blk.data[cw] = <-data
		}
		blk.ecc = ec.calcECC(blk.data, vi.ErrorCorrectionCodewordsPerBlock)
		result[b] = blk
	}

	for b := 0; b < int(vi.NumberOfBlocksInGroup2); b++ {
		blk := new(block)
		blk.data = make([]byte, vi.DataCodeWordsPerBlockInGroup2)
		for cw := 0; cw < int(vi.DataCodeWordsPerBlockInGroup2); cw++ {
			blk.data[cw] = <-data
		}
		blk.ecc = ec.calcECC(blk.data, vi.ErrorCorrectionCodewordsPerBlock)
		result[int(vi.NumberOfBlocksInGroup1)+b] = blk
	}

	return result
}

func (bl blockList) interleave(vi *versionInfo) []byte {
	var maxCodewordCount int
	if vi.DataCodeWordsPerBlockInGroup1 > vi.DataCodeWordsPerBlockInGroup2 {
		maxCodewordCount = int(vi.DataCodeWordsPerBlockInGroup1)
	} else {
		maxCodewordCount = int(vi.DataCodeWordsPerBlockInGroup2)
	}
	resultLen := (vi.DataCodeWordsPerBlockInGroup1+vi.ErrorCorrectionCodewordsPerBlock)*vi.NumberOfBlocksInGroup1 +
		(vi.DataCodeWordsPerBlockInGroup2+vi.ErrorCorrectionCodewordsPerBlock)*vi.NumberOfBlocksInGroup2

	result := make([]byte, 0, resultLen)
	for i := 0; i < maxCodewordCount; i++ {
		for b := 0; b < len(bl); b++ {
			if len(bl[b].data) > i {
				result = append(result, bl[b].data[i])
			}
		}
	}
	for i := 0; i < int(vi.ErrorCorrectionCodewordsPerBlock); i++ {
		for b := 0; b < len(bl); b++ {
			result = append(result, bl[b].ecc[i])
		}
	}
	return result
}

type encodeFn func(content string, eccLevel ErrorCorrectionLevel) (*BitList, *versionInfo, error)

// Encoding mode for QR Codes.
type Encoding byte

const (
	// Auto will choose ths best matching encoding
	Auto Encoding = iota
	// Numeric encoding only encodes numbers [0-9]
	Numeric
	// AlphaNumeric encoding only encodes uppercase letters, numbers and  [Space], $, %, *, +, -, ., /, :
	AlphaNumeric
	// Unicode encoding encodes the string as utf-8
	Unicode
	// only for testing purpose
	unknownEncoding
)

func (e Encoding) getEncoder() encodeFn {
	switch e {
	case Auto:
		return encodeAuto
	case Numeric:
		return encodeNumeric
	case AlphaNumeric:
		return encodeAlphaNumeric
	case Unicode:
		return encodeUnicode
	}
	return nil
}

func (e Encoding) String() string {
	switch e {
	case Auto:
		return "Auto"
	case Numeric:
		return "Numeric"
	case AlphaNumeric:
		return "AlphaNumeric"
	case Unicode:
		return "Unicode"
	}
	return ""
}

// Encode returns a QR barcode with the given content, error correction level and uses the given encoding
func Encode(content string, level ErrorCorrectionLevel, mode Encoding) (Barcode, error) {
	bits, vi, err := mode.getEncoder()(content, level)
	if err != nil {
		return nil, err
	}

	blocks := splitToBlocks(bits.IterateBytes(), vi)
	data := blocks.interleave(vi)
	result := render(data, vi)
	result.content = content
	return result, nil
}

func render(data []byte, vi *versionInfo) *qrcode {
	dim := vi.modulWidth()
	results := make([]*qrcode, 8)
	for i := 0; i < 8; i++ {
		results[i] = newBarcode(dim)
	}

	occupied := newBarcode(dim)

	setAll := func(x, y int, val bool) {
		occupied.Set(x, y, true)
		for i := 0; i < 8; i++ {
			results[i].Set(x, y, val)
		}
	}

	drawFinderPatterns(vi, setAll)
	drawAlignmentPatterns(occupied, vi, setAll)

	// Timing Pattern:
	var i int
	for i = 0; i < dim; i++ {
		if !occupied.Get(i, 6) {
			setAll(i, 6, i%2 == 0)
		}
		if !occupied.Get(6, i) {
			setAll(6, i, i%2 == 0)
		}
	}
	// Dark Module
	setAll(8, dim-8, true)

	drawVersionInfo(vi, setAll)
	drawFormatInfo(vi, -1, occupied.Set)
	for i := 0; i < 8; i++ {
		drawFormatInfo(vi, i, results[i].Set)
	}

	// Write the data
	var curBitNo int

	for pos := range iterateModules(occupied) {
		var curBit bool
		if curBitNo < len(data)*8 {
			curBit = ((data[curBitNo/8] >> uint(7-(curBitNo%8))) & 1) == 1
		} else {
			curBit = false
		}

		for i := 0; i < 8; i++ {
			setMasked(pos.X, pos.Y, curBit, i, results[i].Set)
		}
		curBitNo++
	}

	lowestPenalty := ^uint(0)
	lowestPenaltyIdx := -1
	for i := 0; i < 8; i++ {
		p := results[i].calcPenalty()
		if p < lowestPenalty {
			lowestPenalty = p
			lowestPenaltyIdx = i
		}
	}
	return results[lowestPenaltyIdx]
}

func setMasked(x, y int, val bool, mask int, set func(int, int, bool)) {
	switch mask {
	case 0:
		val = val != (((y + x) % 2) == 0)
	case 1:
		val = val != ((y % 2) == 0)
	case 2:
		val = val != ((x % 3) == 0)
	case 3:
		val = val != (((y + x) % 3) == 0)
	case 4:
		val = val != (((y/2 + x/3) % 2) == 0)
	case 5:
		val = val != (((y*x)%2)+((y*x)%3) == 0)
	case 6:
		val = val != ((((y*x)%2)+((y*x)%3))%2 == 0)
	case 7:
		val = val != ((((y+x)%2)+((y*x)%3))%2 == 0)
	}
	set(x, y, val)
}

func iterateModules(occupied *qrcode) <-chan image.Point {
	result := make(chan image.Point)
	allPoints := make(chan image.Point)
	go func() {
		curX := occupied.dimension - 1
		curY := occupied.dimension - 1
		isUpward := true

		for true {
			if isUpward {
				allPoints <- image.Pt(curX, curY)
				allPoints <- image.Pt(curX-1, curY)
				curY--
				if curY < 0 {
					curY = 0
					curX -= 2
					if curX == 6 {
						curX--
					}
					if curX < 0 {
						break
					}
					isUpward = false
				}
			} else {
				allPoints <- image.Pt(curX, curY)
				allPoints <- image.Pt(curX-1, curY)
				curY++
				if curY >= occupied.dimension {
					curY = occupied.dimension - 1
					curX -= 2
					if curX == 6 {
						curX--
					}
					isUpward = true
					if curX < 0 {
						break
					}
				}
			}
		}

		close(allPoints)
	}()
	go func() {
		for pt := range allPoints {
			if !occupied.Get(pt.X, pt.Y) {
				result <- pt
			}
		}
		close(result)
	}()
	return result
}

func drawFinderPatterns(vi *versionInfo, set func(int, int, bool)) {
	dim := vi.modulWidth()
	drawPattern := func(xoff, yoff int) {
		for x := -1; x < 8; x++ {
			for y := -1; y < 8; y++ {
				val := (x == 0 || x == 6 || y == 0 || y == 6 || (x > 1 && x < 5 && y > 1 && y < 5)) && (x <= 6 && y <= 6 && x >= 0 && y >= 0)

				if x+xoff >= 0 && x+xoff < dim && y+yoff >= 0 && y+yoff < dim {
					set(x+xoff, y+yoff, val)
				}
			}
		}
	}
	drawPattern(0, 0)
	drawPattern(0, dim-7)
	drawPattern(dim-7, 0)
}

func drawAlignmentPatterns(occupied *qrcode, vi *versionInfo, set func(int, int, bool)) {
	drawPattern := func(xoff, yoff int) {
		for x := -2; x <= 2; x++ {
			for y := -2; y <= 2; y++ {
				val := x == -2 || x == 2 || y == -2 || y == 2 || (x == 0 && y == 0)
				set(x+xoff, y+yoff, val)
			}
		}
	}
	positions := vi.alignmentPatternPlacements()

	for _, x := range positions {
		for _, y := range positions {
			if occupied.Get(x, y) {
				continue
			}
			drawPattern(x, y)
		}
	}
}

var formatInfos = map[ErrorCorrectionLevel]map[int][]bool{
	L: {
		0: []bool{true, true, true, false, true, true, true, true, true, false, false, false, true, false, false},
		1: []bool{true, true, true, false, false, true, false, true, true, true, true, false, false, true, true},
		2: []bool{true, true, true, true, true, false, true, true, false, true, false, true, false, true, false},
		3: []bool{true, true, true, true, false, false, false, true, false, false, true, true, true, false, true},
		4: []bool{true, true, false, false, true, true, false, false, false, true, false, true, true, true, true},
		5: []bool{true, true, false, false, false, true, true, false, false, false, true, true, false, false, false},
		6: []bool{true, true, false, true, true, false, false, false, true, false, false, false, false, false, true},
		7: []bool{true, true, false, true, false, false, true, false, true, true, true, false, true, true, false},
	},
	M: {
		0: []bool{true, false, true, false, true, false, false, false, false, false, true, false, false, true, false},
		1: []bool{true, false, true, false, false, false, true, false, false, true, false, false, true, false, true},
		2: []bool{true, false, true, true, true, true, false, false, true, true, true, true, true, false, false},
		3: []bool{true, false, true, true, false, true, true, false, true, false, false, true, false, true, true},
		4: []bool{true, false, false, false, true, false, true, true, true, true, true, true, false, false, true},
		5: []bool{true, false, false, false, false, false, false, true, true, false, false, true, true, true, false},
		6: []bool{true, false, false, true, true, true, true, true, false, false, true, false, true, true, true},
		7: []bool{true, false, false, true, false, true, false, true, false, true, false, false, false, false, false},
	},
	Q: {
		0: []bool{false, true, true, false, true, false, true, false, true, false, true, true, true, true, true},
		1: []bool{false, true, true, false, false, false, false, false, true, true, false, true, false, false, false},
		2: []bool{false, true, true, true, true, true, true, false, false, true, true, false, false, false, true},
		3: []bool{false, true, true, true, false, true, false, false, false, false, false, false, true, true, false},
		4: []bool{false, true, false, false, true, false, false, true, false, true, true, false, true, false, false},
		5: []bool{false, true, false, false, false, false, true, true, false, false, false, false, false, true, true},
		6: []bool{false, true, false, true, true, true, false, true, true, false, true, true, false, true, false},
		7: []bool{false, true, false, true, false, true, true, true, true, true, false, true, true, false, true},
	},
	H: {
		0: []bool{false, false, true, false, true, true, false, true, false, false, false, true, false, false, true},
		1: []bool{false, false, true, false, false, true, true, true, false, true, true, true, true, true, false},
		2: []bool{false, false, true, true, true, false, false, true, true, true, false, false, true, true, true},
		3: []bool{false, false, true, true, false, false, true, true, true, false, true, false, false, false, false},
		4: []bool{false, false, false, false, true, true, true, false, true, true, false, false, false, true, false},
		5: []bool{false, false, false, false, false, true, false, false, true, false, true, false, true, false, true},
		6: []bool{false, false, false, true, true, false, true, false, false, false, false, true, true, false, false},
		7: []bool{false, false, false, true, false, false, false, false, false, true, true, true, false, true, true},
	},
}

func drawFormatInfo(vi *versionInfo, usedMask int, set func(int, int, bool)) {
	var formatInfo []bool

	if usedMask == -1 {
		formatInfo = []bool{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true} // Set all to true cause -1 --> occupied mask.
	} else {
		formatInfo = formatInfos[vi.Level][usedMask]
	}

	if len(formatInfo) == 15 {
		dim := vi.modulWidth()
		set(0, 8, formatInfo[0])
		set(1, 8, formatInfo[1])
		set(2, 8, formatInfo[2])
		set(3, 8, formatInfo[3])
		set(4, 8, formatInfo[4])
		set(5, 8, formatInfo[5])
		set(7, 8, formatInfo[6])
		set(8, 8, formatInfo[7])
		set(8, 7, formatInfo[8])
		set(8, 5, formatInfo[9])
		set(8, 4, formatInfo[10])
		set(8, 3, formatInfo[11])
		set(8, 2, formatInfo[12])
		set(8, 1, formatInfo[13])
		set(8, 0, formatInfo[14])

		set(8, dim-1, formatInfo[0])
		set(8, dim-2, formatInfo[1])
		set(8, dim-3, formatInfo[2])
		set(8, dim-4, formatInfo[3])
		set(8, dim-5, formatInfo[4])
		set(8, dim-6, formatInfo[5])
		set(8, dim-7, formatInfo[6])
		set(dim-8, 8, formatInfo[7])
		set(dim-7, 8, formatInfo[8])
		set(dim-6, 8, formatInfo[9])
		set(dim-5, 8, formatInfo[10])
		set(dim-4, 8, formatInfo[11])
		set(dim-3, 8, formatInfo[12])
		set(dim-2, 8, formatInfo[13])
		set(dim-1, 8, formatInfo[14])
	}
}

var versionInfoBitsByVersion = map[byte][]bool{
	7:  {false, false, false, true, true, true, true, true, false, false, true, false, false, true, false, true, false, false},
	8:  {false, false, true, false, false, false, false, true, false, true, true, false, true, true, true, true, false, false},
	9:  {false, false, true, false, false, true, true, false, true, false, true, false, false, true, true, false, false, true},
	10: {false, false, true, false, true, false, false, true, false, false, true, true, false, true, false, false, true, true},
	11: {false, false, true, false, true, true, true, false, true, true, true, true, true, true, false, true, true, false},
	12: {false, false, true, true, false, false, false, true, true, true, false, true, true, false, false, false, true, false},
	13: {false, false, true, true, false, true, true, false, false, false, false, true, false, false, false, true, true, true},
	14: {false, false, true, true, true, false, false, true, true, false, false, false, false, false, true, true, false, true},
	15: {false, false, true, true, true, true, true, false, false, true, false, false, true, false, true, false, false, false},
	16: {false, true, false, false, false, false, true, false, true, true, false, true, true, true, true, false, false, false},
	17: {false, true, false, false, false, true, false, true, false, false, false, true, false, true, true, true, false, true},
	18: {false, true, false, false, true, false, true, false, true, false, false, false, false, true, false, true, true, true},
	19: {false, true, false, false, true, true, false, true, false, true, false, false, true, true, false, false, true, false},
	20: {false, true, false, true, false, false, true, false, false, true, true, false, true, false, false, true, true, false},
	21: {false, true, false, true, false, true, false, true, true, false, true, false, false, false, false, false, true, true},
	22: {false, true, false, true, true, false, true, false, false, false, true, true, false, false, true, false, false, true},
	23: {false, true, false, true, true, true, false, true, true, true, true, true, true, false, true, true, false, false},
	24: {false, true, true, false, false, false, true, true, true, false, true, true, false, false, false, true, false, false},
	25: {false, true, true, false, false, true, false, false, false, true, true, true, true, false, false, false, false, true},
	26: {false, true, true, false, true, false, true, true, true, true, true, false, true, false, true, false, true, true},
	27: {false, true, true, false, true, true, false, false, false, false, true, false, false, false, true, true, true, false},
	28: {false, true, true, true, false, false, true, true, false, false, false, false, false, true, true, false, true, false},
	29: {false, true, true, true, false, true, false, false, true, true, false, false, true, true, true, true, true, true},
	30: {false, true, true, true, true, false, true, true, false, true, false, true, true, true, false, true, false, true},
	31: {false, true, true, true, true, true, false, false, true, false, false, true, false, true, false, false, false, false},
	32: {true, false, false, false, false, false, true, false, false, true, true, true, false, true, false, true, false, true},
	33: {true, false, false, false, false, true, false, true, true, false, true, true, true, true, false, false, false, false},
	34: {true, false, false, false, true, false, true, false, false, false, true, false, true, true, true, false, true, false},
	35: {true, false, false, false, true, true, false, true, true, true, true, false, false, true, true, true, true, true},
	36: {true, false, false, true, false, false, true, false, true, true, false, false, false, false, true, false, true, true},
	37: {true, false, false, true, false, true, false, true, false, false, false, false, true, false, true, true, true, false},
	38: {true, false, false, true, true, false, true, false, true, false, false, true, true, false, false, true, false, false},
	39: {true, false, false, true, true, true, false, true, false, true, false, true, false, false, false, false, false, true},
	40: {true, false, true, false, false, false, true, true, false, false, false, true, true, false, true, false, false, true},
}

func drawVersionInfo(vi *versionInfo, set func(int, int, bool)) {
	versionInfoBits, ok := versionInfoBitsByVersion[vi.Version]

	if ok && len(versionInfoBits) > 0 {
		for i := 0; i < len(versionInfoBits); i++ {
			x := (vi.modulWidth() - 11) + i%3
			y := i / 3
			set(x, y, versionInfoBits[len(versionInfoBits)-i-1])
			set(y, x, versionInfoBits[len(versionInfoBits)-i-1])
		}
	}
}

func addPaddingAndTerminator(bl *BitList, vi *versionInfo) {
	for i := 0; i < 4 && bl.Len() < vi.totalDataBytes()*8; i++ {
		bl.AddBit(false)
	}

	for bl.Len()%8 != 0 {
		bl.AddBit(false)
	}

	for i := 0; bl.Len() < vi.totalDataBytes()*8; i++ {
		if i%2 == 0 {
			bl.AddByte(236)
		} else {
			bl.AddByte(17)
		}
	}
}

type errorCorrection struct {
	rs *ReedSolomonEncoder
}

var ec = newErrorCorrection()

func newErrorCorrection() *errorCorrection {
	fld := NewGaloisField(285, 256, 0)
	return &errorCorrection{NewReedSolomonEncoder(fld)}
}

func (ec *errorCorrection) calcECC(data []byte, eccCount byte) []byte {
	dataInts := make([]int, len(data))
	for i := 0; i < len(data); i++ {
		dataInts[i] = int(data[i])
	}
	res := ec.rs.Encode(dataInts, int(eccCount))
	result := make([]byte, len(res))
	for i := 0; i < len(res); i++ {
		result[i] = byte(res[i])
	}
	return result
}

func encodeNumeric(content string, ecl ErrorCorrectionLevel) (*BitList, *versionInfo, error) {
	contentBitCount := (len(content) / 3) * 10
	switch len(content) % 3 {
	case 1:
		contentBitCount += 4
	case 2:
		contentBitCount += 7
	}
	vi := findSmallestVersionInfo(ecl, numericMode, contentBitCount)
	if vi == nil {
		return nil, nil, errors.New("To much data to encode")
	}
	res := new(BitList)
	res.AddBits(int(numericMode), 4)
	res.AddBits(len(content), vi.charCountBits(numericMode))

	for pos := 0; pos < len(content); pos += 3 {
		var curStr string
		if pos+3 <= len(content) {
			curStr = content[pos : pos+3]
		} else {
			curStr = content[pos:]
		}

		i, err := strconv.Atoi(curStr)
		if err != nil || i < 0 {
			return nil, nil, fmt.Errorf("\"%s\" can not be encoded as %s", content, Numeric)
		}
		var bitCnt byte
		switch len(curStr) % 3 {
		case 0:
			bitCnt = 10
		case 1:
			bitCnt = 4
		case 2:
			bitCnt = 7
		}

		res.AddBits(i, bitCnt)
	}

	addPaddingAndTerminator(res, vi)
	return res, vi, nil
}

type qrcode struct {
	dimension int
	data      *BitList
	content   string
}

func (qr *qrcode) Content() string {
	return qr.content
}

func (qr *qrcode) Metadata() Metadata {
	return Metadata{TypeQR, 2}
}

func (qr *qrcode) ColorModel() color.Model {
	return color.Gray16Model
}

func (qr *qrcode) Bounds() image.Rectangle {
	return image.Rect(0, 0, qr.dimension, qr.dimension)
}

func (qr *qrcode) At(x, y int) color.Color {
	if qr.Get(x, y) {
		return color.Black
	}
	return color.White
}

func (qr *qrcode) Get(x, y int) bool {
	return qr.data.GetBit(x*qr.dimension + y)
}

func (qr *qrcode) Set(x, y int, val bool) {
	qr.data.SetBit(x*qr.dimension+y, val)
}

func (qr *qrcode) calcPenalty() uint {
	return qr.calcPenaltyRule1() + qr.calcPenaltyRule2() + qr.calcPenaltyRule3() + qr.calcPenaltyRule4()
}

func (qr *qrcode) calcPenaltyRule1() uint {
	var result uint
	for x := 0; x < qr.dimension; x++ {
		checkForX := false
		var cntX uint
		checkForY := false
		var cntY uint

		for y := 0; y < qr.dimension; y++ {
			if qr.Get(x, y) == checkForX {
				cntX++
			} else {
				checkForX = !checkForX
				if cntX >= 5 {
					result += cntX - 2
				}
				cntX = 1
			}

			if qr.Get(y, x) == checkForY {
				cntY++
			} else {
				checkForY = !checkForY
				if cntY >= 5 {
					result += cntY - 2
				}
				cntY = 1
			}
		}

		if cntX >= 5 {
			result += cntX - 2
		}
		if cntY >= 5 {
			result += cntY - 2
		}
	}

	return result
}

func (qr *qrcode) calcPenaltyRule2() uint {
	var result uint
	for x := 0; x < qr.dimension-1; x++ {
		for y := 0; y < qr.dimension-1; y++ {
			check := qr.Get(x, y)
			if qr.Get(x, y+1) == check && qr.Get(x+1, y) == check && qr.Get(x+1, y+1) == check {
				result += 3
			}
		}
	}
	return result
}

func (qr *qrcode) calcPenaltyRule3() uint {
	pattern1 := []bool{true, false, true, true, true, false, true, false, false, false, false}
	pattern2 := []bool{false, false, false, false, true, false, true, true, true, false, true}

	var result uint
	for x := 0; x <= qr.dimension-len(pattern1); x++ {
		for y := 0; y < qr.dimension; y++ {
			pattern1XFound := true
			pattern2XFound := true
			pattern1YFound := true
			pattern2YFound := true

			for i := 0; i < len(pattern1); i++ {
				iv := qr.Get(x+i, y)
				if iv != pattern1[i] {
					pattern1XFound = false
				}
				if iv != pattern2[i] {
					pattern2XFound = false
				}
				iv = qr.Get(y, x+i)
				if iv != pattern1[i] {
					pattern1YFound = false
				}
				if iv != pattern2[i] {
					pattern2YFound = false
				}
			}
			if pattern1XFound || pattern2XFound {
				result += 40
			}
			if pattern1YFound || pattern2YFound {
				result += 40
			}
		}
	}

	return result
}

func (qr *qrcode) calcPenaltyRule4() uint {
	totalNum := qr.data.Len()
	trueCnt := 0
	for i := 0; i < totalNum; i++ {
		if qr.data.GetBit(i) {
			trueCnt++
		}
	}
	percDark := float64(trueCnt) * 100 / float64(totalNum)
	floor := math.Abs(math.Floor(percDark/5) - 10)
	ceil := math.Abs(math.Ceil(percDark/5) - 10)
	return uint(math.Min(floor, ceil) * 10)
}

func newBarcode(dim int) *qrcode {
	res := new(qrcode)
	res.dimension = dim
	res.data = NewBitList(dim * dim)
	return res
}

func encodeUnicode(content string, ecl ErrorCorrectionLevel) (*BitList, *versionInfo, error) {
	data := []byte(content)

	vi := findSmallestVersionInfo(ecl, byteMode, len(data)*8)
	if vi == nil {
		return nil, nil, errors.New("To much data to encode")
	}

	// It's not correct to add the unicode bytes to the result directly but most readers can't handle the
	// required ECI header...
	res := new(BitList)
	res.AddBits(int(byteMode), 4)
	res.AddBits(len(content), vi.charCountBits(byteMode))
	for _, b := range data {
		res.AddByte(b)
	}
	addPaddingAndTerminator(res, vi)
	return res, vi, nil
}

// ErrorCorrectionLevel indicates the amount of "backup data" stored in the QR code
type ErrorCorrectionLevel byte

const (
	// L recovers 7% of data
	L ErrorCorrectionLevel = iota
	// M recovers 15% of data
	M
	// Q recovers 25% of data
	Q
	// H recovers 30% of data
	H
)

func (ecl ErrorCorrectionLevel) String() string {
	switch ecl {
	case L:
		return "L"
	case M:
		return "M"
	case Q:
		return "Q"
	case H:
		return "H"
	}
	return "unknown"
}

type encodingMode byte

const (
	numericMode      encodingMode = 1
	alphaNumericMode encodingMode = 2
	byteMode         encodingMode = 4
	kanjiMode        encodingMode = 8
)

type versionInfo struct {
	Version                          byte
	Level                            ErrorCorrectionLevel
	ErrorCorrectionCodewordsPerBlock byte
	NumberOfBlocksInGroup1           byte
	DataCodeWordsPerBlockInGroup1    byte
	NumberOfBlocksInGroup2           byte
	DataCodeWordsPerBlockInGroup2    byte
}

var versionInfos = []*versionInfo{
	{1, L, 7, 1, 19, 0, 0},
	{1, M, 10, 1, 16, 0, 0},
	{1, Q, 13, 1, 13, 0, 0},
	{1, H, 17, 1, 9, 0, 0},
	{2, L, 10, 1, 34, 0, 0},
	{2, M, 16, 1, 28, 0, 0},
	{2, Q, 22, 1, 22, 0, 0},
	{2, H, 28, 1, 16, 0, 0},
	{3, L, 15, 1, 55, 0, 0},
	{3, M, 26, 1, 44, 0, 0},
	{3, Q, 18, 2, 17, 0, 0},
	{3, H, 22, 2, 13, 0, 0},
	{4, L, 20, 1, 80, 0, 0},
	{4, M, 18, 2, 32, 0, 0},
	{4, Q, 26, 2, 24, 0, 0},
	{4, H, 16, 4, 9, 0, 0},
	{5, L, 26, 1, 108, 0, 0},
	{5, M, 24, 2, 43, 0, 0},
	{5, Q, 18, 2, 15, 2, 16},
	{5, H, 22, 2, 11, 2, 12},
	{6, L, 18, 2, 68, 0, 0},
	{6, M, 16, 4, 27, 0, 0},
	{6, Q, 24, 4, 19, 0, 0},
	{6, H, 28, 4, 15, 0, 0},
	{7, L, 20, 2, 78, 0, 0},
	{7, M, 18, 4, 31, 0, 0},
	{7, Q, 18, 2, 14, 4, 15},
	{7, H, 26, 4, 13, 1, 14},
	{8, L, 24, 2, 97, 0, 0},
	{8, M, 22, 2, 38, 2, 39},
	{8, Q, 22, 4, 18, 2, 19},
	{8, H, 26, 4, 14, 2, 15},
	{9, L, 30, 2, 116, 0, 0},
	{9, M, 22, 3, 36, 2, 37},
	{9, Q, 20, 4, 16, 4, 17},
	{9, H, 24, 4, 12, 4, 13},
	{10, L, 18, 2, 68, 2, 69},
	{10, M, 26, 4, 43, 1, 44},
	{10, Q, 24, 6, 19, 2, 20},
	{10, H, 28, 6, 15, 2, 16},
	{11, L, 20, 4, 81, 0, 0},
	{11, M, 30, 1, 50, 4, 51},
	{11, Q, 28, 4, 22, 4, 23},
	{11, H, 24, 3, 12, 8, 13},
	{12, L, 24, 2, 92, 2, 93},
	{12, M, 22, 6, 36, 2, 37},
	{12, Q, 26, 4, 20, 6, 21},
	{12, H, 28, 7, 14, 4, 15},
	{13, L, 26, 4, 107, 0, 0},
	{13, M, 22, 8, 37, 1, 38},
	{13, Q, 24, 8, 20, 4, 21},
	{13, H, 22, 12, 11, 4, 12},
	{14, L, 30, 3, 115, 1, 116},
	{14, M, 24, 4, 40, 5, 41},
	{14, Q, 20, 11, 16, 5, 17},
	{14, H, 24, 11, 12, 5, 13},
	{15, L, 22, 5, 87, 1, 88},
	{15, M, 24, 5, 41, 5, 42},
	{15, Q, 30, 5, 24, 7, 25},
	{15, H, 24, 11, 12, 7, 13},
	{16, L, 24, 5, 98, 1, 99},
	{16, M, 28, 7, 45, 3, 46},
	{16, Q, 24, 15, 19, 2, 20},
	{16, H, 30, 3, 15, 13, 16},
	{17, L, 28, 1, 107, 5, 108},
	{17, M, 28, 10, 46, 1, 47},
	{17, Q, 28, 1, 22, 15, 23},
	{17, H, 28, 2, 14, 17, 15},
	{18, L, 30, 5, 120, 1, 121},
	{18, M, 26, 9, 43, 4, 44},
	{18, Q, 28, 17, 22, 1, 23},
	{18, H, 28, 2, 14, 19, 15},
	{19, L, 28, 3, 113, 4, 114},
	{19, M, 26, 3, 44, 11, 45},
	{19, Q, 26, 17, 21, 4, 22},
	{19, H, 26, 9, 13, 16, 14},
	{20, L, 28, 3, 107, 5, 108},
	{20, M, 26, 3, 41, 13, 42},
	{20, Q, 30, 15, 24, 5, 25},
	{20, H, 28, 15, 15, 10, 16},
	{21, L, 28, 4, 116, 4, 117},
	{21, M, 26, 17, 42, 0, 0},
	{21, Q, 28, 17, 22, 6, 23},
	{21, H, 30, 19, 16, 6, 17},
	{22, L, 28, 2, 111, 7, 112},
	{22, M, 28, 17, 46, 0, 0},
	{22, Q, 30, 7, 24, 16, 25},
	{22, H, 24, 34, 13, 0, 0},
	{23, L, 30, 4, 121, 5, 122},
	{23, M, 28, 4, 47, 14, 48},
	{23, Q, 30, 11, 24, 14, 25},
	{23, H, 30, 16, 15, 14, 16},
	{24, L, 30, 6, 117, 4, 118},
	{24, M, 28, 6, 45, 14, 46},
	{24, Q, 30, 11, 24, 16, 25},
	{24, H, 30, 30, 16, 2, 17},
	{25, L, 26, 8, 106, 4, 107},
	{25, M, 28, 8, 47, 13, 48},
	{25, Q, 30, 7, 24, 22, 25},
	{25, H, 30, 22, 15, 13, 16},
	{26, L, 28, 10, 114, 2, 115},
	{26, M, 28, 19, 46, 4, 47},
	{26, Q, 28, 28, 22, 6, 23},
	{26, H, 30, 33, 16, 4, 17},
	{27, L, 30, 8, 122, 4, 123},
	{27, M, 28, 22, 45, 3, 46},
	{27, Q, 30, 8, 23, 26, 24},
	{27, H, 30, 12, 15, 28, 16},
	{28, L, 30, 3, 117, 10, 118},
	{28, M, 28, 3, 45, 23, 46},
	{28, Q, 30, 4, 24, 31, 25},
	{28, H, 30, 11, 15, 31, 16},
	{29, L, 30, 7, 116, 7, 117},
	{29, M, 28, 21, 45, 7, 46},
	{29, Q, 30, 1, 23, 37, 24},
	{29, H, 30, 19, 15, 26, 16},
	{30, L, 30, 5, 115, 10, 116},
	{30, M, 28, 19, 47, 10, 48},
	{30, Q, 30, 15, 24, 25, 25},
	{30, H, 30, 23, 15, 25, 16},
	{31, L, 30, 13, 115, 3, 116},
	{31, M, 28, 2, 46, 29, 47},
	{31, Q, 30, 42, 24, 1, 25},
	{31, H, 30, 23, 15, 28, 16},
	{32, L, 30, 17, 115, 0, 0},
	{32, M, 28, 10, 46, 23, 47},
	{32, Q, 30, 10, 24, 35, 25},
	{32, H, 30, 19, 15, 35, 16},
	{33, L, 30, 17, 115, 1, 116},
	{33, M, 28, 14, 46, 21, 47},
	{33, Q, 30, 29, 24, 19, 25},
	{33, H, 30, 11, 15, 46, 16},
	{34, L, 30, 13, 115, 6, 116},
	{34, M, 28, 14, 46, 23, 47},
	{34, Q, 30, 44, 24, 7, 25},
	{34, H, 30, 59, 16, 1, 17},
	{35, L, 30, 12, 121, 7, 122},
	{35, M, 28, 12, 47, 26, 48},
	{35, Q, 30, 39, 24, 14, 25},
	{35, H, 30, 22, 15, 41, 16},
	{36, L, 30, 6, 121, 14, 122},
	{36, M, 28, 6, 47, 34, 48},
	{36, Q, 30, 46, 24, 10, 25},
	{36, H, 30, 2, 15, 64, 16},
	{37, L, 30, 17, 122, 4, 123},
	{37, M, 28, 29, 46, 14, 47},
	{37, Q, 30, 49, 24, 10, 25},
	{37, H, 30, 24, 15, 46, 16},
	{38, L, 30, 4, 122, 18, 123},
	{38, M, 28, 13, 46, 32, 47},
	{38, Q, 30, 48, 24, 14, 25},
	{38, H, 30, 42, 15, 32, 16},
	{39, L, 30, 20, 117, 4, 118},
	{39, M, 28, 40, 47, 7, 48},
	{39, Q, 30, 43, 24, 22, 25},
	{39, H, 30, 10, 15, 67, 16},
	{40, L, 30, 19, 118, 6, 119},
	{40, M, 28, 18, 47, 31, 48},
	{40, Q, 30, 34, 24, 34, 25},
	{40, H, 30, 20, 15, 61, 16},
}

func (vi *versionInfo) totalDataBytes() int {
	g1Data := int(vi.NumberOfBlocksInGroup1) * int(vi.DataCodeWordsPerBlockInGroup1)
	g2Data := int(vi.NumberOfBlocksInGroup2) * int(vi.DataCodeWordsPerBlockInGroup2)
	return (g1Data + g2Data)
}

func (vi *versionInfo) charCountBits(m encodingMode) byte {
	switch m {
	case numericMode:
		if vi.Version < 10 {
			return 10
		} else if vi.Version < 27 {
			return 12
		}
		return 14

	case alphaNumericMode:
		if vi.Version < 10 {
			return 9
		} else if vi.Version < 27 {
			return 11
		}
		return 13

	case byteMode:
		if vi.Version < 10 {
			return 8
		}
		return 16

	case kanjiMode:
		if vi.Version < 10 {
			return 8
		} else if vi.Version < 27 {
			return 10
		}
		return 12
	default:
		return 0
	}
}

func (vi *versionInfo) modulWidth() int {
	return ((int(vi.Version) - 1) * 4) + 21
}

func (vi *versionInfo) alignmentPatternPlacements() []int {
	if vi.Version == 1 {
		return make([]int, 0)
	}

	first := 6
	last := vi.modulWidth() - 7
	space := float64(last - first)
	count := int(math.Ceil(space/28)) + 1

	result := make([]int, count)
	result[0] = first
	result[len(result)-1] = last
	if count > 2 {
		step := int(math.Ceil(float64(last-first) / float64(count-1)))
		if step%2 == 1 {
			frac := float64(last-first) / float64(count-1)
			_, x := math.Modf(frac)
			if x >= 0.5 {
				frac = math.Ceil(frac)
			} else {
				frac = math.Floor(frac)
			}

			if int(frac)%2 == 0 {
				step--
			} else {
				step++
			}
		}

		for i := 1; i <= count-2; i++ {
			result[i] = last - (step * (count - 1 - i))
		}
	}

	return result
}

func findSmallestVersionInfo(ecl ErrorCorrectionLevel, mode encodingMode, dataBits int) *versionInfo {
	dataBits = dataBits + 4 // mode indicator
	for _, vi := range versionInfos {
		if vi.Level == ecl {
			if (vi.totalDataBytes() * 8) >= (dataBits + int(vi.charCountBits(mode))) {
				return vi
			}
		}
	}
	return nil
}

// Package utils contain some utilities which are needed to create barcodes

type base1DCode struct {
	*BitList
	kind    string
	content string
}

type base1DCodeIntCS struct {
	base1DCode
	checksum int
}

func (c *base1DCode) Content() string {
	return c.content
}

func (c *base1DCode) Metadata() Metadata {
	return Metadata{c.kind, 1}
}

func (c *base1DCode) ColorModel() color.Model {
	return color.Gray16Model
}

func (c *base1DCode) Bounds() image.Rectangle {
	return image.Rect(0, 0, c.Len(), 1)
}

// At ...
func (c *base1DCode) At(x, _ int) color.Color {
	if c.GetBit(x) {
		return color.Black
	}
	return color.White
}

// CheckSum ...
func (c *base1DCodeIntCS) CheckSum() int {
	return c.checksum
}

// New1DCodeIntCheckSum creates a new 1D barcode where the bars are represented by the bits in the bars BitList
func New1DCodeIntCheckSum(codeKind, content string, bars *BitList, checksum int) BarcodeIntCS {
	return &base1DCodeIntCS{base1DCode{bars, codeKind, content}, checksum}
}

// New1DCode creates a new 1D barcode where the bars are represented by the bits in the bars BitList
func New1DCode(codeKind, content string, bars *BitList) Barcode {
	return &base1DCode{bars, codeKind, content}
}

// BitList is a list that contains bits
type BitList struct {
	count int
	data  []int32
}

// NewBitList returns a new BitList with the given length
// all bits are initialize with false
func NewBitList(capacity int) *BitList {
	bl := new(BitList)
	bl.count = capacity
	x := 0
	if capacity%32 != 0 {
		x = 1
	}
	bl.data = make([]int32, capacity/32+x)
	return bl
}

// Len returns the number of contained bits
func (bl *BitList) Len() int {
	return bl.count
}

func (bl *BitList) grow() {
	growBy := len(bl.data)
	if growBy < 128 {
		growBy = 128
	} else if growBy >= 1024 {
		growBy = 1024
	}

	nd := make([]int32, len(bl.data)+growBy)
	copy(nd, bl.data)
	bl.data = nd
}

// AddBit appends the given bits to the end of the list
func (bl *BitList) AddBit(bits ...bool) {
	for _, bit := range bits {
		itmIndex := bl.count / 32
		for itmIndex >= len(bl.data) {
			bl.grow()
		}
		bl.SetBit(bl.count, bit)
		bl.count++
	}
}

// SetBit sets the bit at the given index to the given value
func (bl *BitList) SetBit(index int, value bool) {
	itmIndex := index / 32
	itmBitShift := 31 - (index % 32)
	if value {
		bl.data[itmIndex] = bl.data[itmIndex] | 1<<uint(itmBitShift)
	} else {
		bl.data[itmIndex] = bl.data[itmIndex] & ^(1 << uint(itmBitShift))
	}
}

// GetBit returns the bit at the given index
func (bl *BitList) GetBit(index int) bool {
	itmIndex := index / 32
	itmBitShift := 31 - (index % 32)
	return ((bl.data[itmIndex] >> uint(itmBitShift)) & 1) == 1
}

// AddByte appends all 8 bits of the given byte to the end of the list
func (bl *BitList) AddByte(b byte) {
	for i := 7; i >= 0; i-- {
		bl.AddBit(((b >> uint(i)) & 1) == 1)
	}
}

// AddBits appends the last (LSB) 'count' bits of 'b' the the end of the list
func (bl *BitList) AddBits(b int, count byte) {
	for i := int(count) - 1; i >= 0; i-- {
		bl.AddBit(((b >> uint(i)) & 1) == 1)
	}
}

// GetBytes returns all bits of the BitList as a []byte
func (bl *BitList) GetBytes() []byte {
	len := bl.count >> 3
	if (bl.count % 8) != 0 {
		len++
	}
	result := make([]byte, len)
	for i := 0; i < len; i++ {
		shift := (3 - (i % 4)) * 8
		result[i] = (byte)((bl.data[i/4] >> uint(shift)) & 0xFF)
	}
	return result
}

// IterateBytes iterates through all bytes contained in the BitList
func (bl *BitList) IterateBytes() <-chan byte {
	res := make(chan byte)

	go func() {
		c := bl.count
		shift := 24
		i := 0
		for c > 0 {
			res <- byte((bl.data[i] >> uint(shift)) & 0xFF)
			shift -= 8
			if shift < 0 {
				shift = 24
				i++
			}
			c -= 8
		}
		close(res)
	}()

	return res
}

// GaloisField encapsulates galois field arithmetics
type GaloisField struct {
	Size    int
	Base    int
	ALogTbl []int
	LogTbl  []int
}

// NewGaloisField creates a new galois field
func NewGaloisField(pp, fieldSize, b int) *GaloisField {
	result := new(GaloisField)

	result.Size = fieldSize
	result.Base = b
	result.ALogTbl = make([]int, fieldSize)
	result.LogTbl = make([]int, fieldSize)

	x := 1
	for i := 0; i < fieldSize; i++ {
		result.ALogTbl[i] = x
		x = x * 2
		if x >= fieldSize {
			x = (x ^ pp) & (fieldSize - 1)
		}
	}

	for i := 0; i < fieldSize; i++ {
		result.LogTbl[result.ALogTbl[i]] = int(i)
	}

	return result
}

// Zero ...
func (gf *GaloisField) Zero() *GFPoly {
	return NewGFPoly(gf, []int{0})
}

// AddOrSub add or subtract two numbers
func (gf *GaloisField) AddOrSub(a, b int) int {
	return a ^ b
}

// Multiply multiplys two numbers
func (gf *GaloisField) Multiply(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	return gf.ALogTbl[(gf.LogTbl[a]+gf.LogTbl[b])%(gf.Size-1)]
}

// Divide divides two numbers
func (gf *GaloisField) Divide(a, b int) int {
	if b == 0 {
		panic("divide by zero")
	} else if a == 0 {
		return 0
	}
	return gf.ALogTbl[(gf.LogTbl[a]-gf.LogTbl[b])%(gf.Size-1)]
}

// Invers ...
func (gf *GaloisField) Invers(num int) int {
	return gf.ALogTbl[(gf.Size-1)-gf.LogTbl[num]]
}

// GFPoly ...
type GFPoly struct {
	gf           *GaloisField
	Coefficients []int
}

// Degree ..
func (gp *GFPoly) Degree() int {
	return len(gp.Coefficients) - 1
}

// Zero ...
func (gp *GFPoly) Zero() bool {
	return gp.Coefficients[0] == 0
}

// GetCoefficient returns the coefficient of x ^ degree
func (gp *GFPoly) GetCoefficient(degree int) int {
	return gp.Coefficients[gp.Degree()-degree]
}

// AddOrSubstract ...
func (gp *GFPoly) AddOrSubstract(other *GFPoly) *GFPoly {
	if gp.Zero() {
		return other
	} else if other.Zero() {
		return gp
	}
	smallCoeff := gp.Coefficients
	largeCoeff := other.Coefficients
	if len(smallCoeff) > len(largeCoeff) {
		largeCoeff, smallCoeff = smallCoeff, largeCoeff
	}
	sumDiff := make([]int, len(largeCoeff))
	lenDiff := len(largeCoeff) - len(smallCoeff)
	copy(sumDiff, largeCoeff[:lenDiff])
	for i := lenDiff; i < len(largeCoeff); i++ {
		sumDiff[i] = int(gp.gf.AddOrSub(int(smallCoeff[i-lenDiff]), int(largeCoeff[i])))
	}
	return NewGFPoly(gp.gf, sumDiff)
}

// MultByMonominal ...
func (gp *GFPoly) MultByMonominal(degree, coeff int) *GFPoly {
	if coeff == 0 {
		return gp.gf.Zero()
	}
	size := len(gp.Coefficients)
	result := make([]int, size+degree)
	for i := 0; i < size; i++ {
		result[i] = int(gp.gf.Multiply(int(gp.Coefficients[i]), int(coeff)))
	}
	return NewGFPoly(gp.gf, result)
}

// Multiply ...
func (gp *GFPoly) Multiply(other *GFPoly) *GFPoly {
	if gp.Zero() || other.Zero() {
		return gp.gf.Zero()
	}
	aCoeff := gp.Coefficients
	aLen := len(aCoeff)
	bCoeff := other.Coefficients
	bLen := len(bCoeff)
	product := make([]int, aLen+bLen-1)
	for i := 0; i < aLen; i++ {
		ac := int(aCoeff[i])
		for j := 0; j < bLen; j++ {
			bc := int(bCoeff[j])
			product[i+j] = int(gp.gf.AddOrSub(int(product[i+j]), gp.gf.Multiply(ac, bc)))
		}
	}
	return NewGFPoly(gp.gf, product)
}

// Divide ...
func (gp *GFPoly) Divide(other *GFPoly) (quotient, remainder *GFPoly) {
	quotient = gp.gf.Zero()
	remainder = gp
	fld := gp.gf
	denomLeadTerm := other.GetCoefficient(other.Degree())
	inversDenomLeadTerm := fld.Invers(int(denomLeadTerm))
	for remainder.Degree() >= other.Degree() && !remainder.Zero() {
		degreeDiff := remainder.Degree() - other.Degree()
		scale := int(fld.Multiply(int(remainder.GetCoefficient(remainder.Degree())), inversDenomLeadTerm))
		term := other.MultByMonominal(degreeDiff, scale)
		itQuot := NewMonominalPoly(fld, degreeDiff, scale)
		quotient = quotient.AddOrSubstract(itQuot)
		remainder = remainder.AddOrSubstract(term)
	}
	return quotient, remainder
}

// NewMonominalPoly ...
func NewMonominalPoly(field *GaloisField, degree, coeff int) *GFPoly {
	if coeff == 0 {
		return field.Zero()
	}
	result := make([]int, degree+1)
	result[0] = coeff
	return NewGFPoly(field, result)
}

// NewGFPoly ...
func NewGFPoly(field *GaloisField, coefficients []int) *GFPoly {
	for len(coefficients) > 1 && coefficients[0] == 0 {
		coefficients = coefficients[1:]
	}
	return &GFPoly{field, coefficients}
}

// ReedSolomonEncoder ...
type ReedSolomonEncoder struct {
	gf        *GaloisField
	polynomes []*GFPoly
	m         *sync.Mutex
}

// NewReedSolomonEncoder ...
func NewReedSolomonEncoder(gf *GaloisField) *ReedSolomonEncoder {
	return &ReedSolomonEncoder{
		gf, []*GFPoly{NewGFPoly(gf, []int{1})}, new(sync.Mutex),
	}
}

func (rs *ReedSolomonEncoder) getPolynomial(degree int) *GFPoly {
	rs.m.Lock()
	defer rs.m.Unlock()

	if degree >= len(rs.polynomes) {
		last := rs.polynomes[len(rs.polynomes)-1]
		for d := len(rs.polynomes); d <= degree; d++ {
			next := last.Multiply(NewGFPoly(rs.gf, []int{1, rs.gf.ALogTbl[d-1+rs.gf.Base]}))
			rs.polynomes = append(rs.polynomes, next)
			last = next
		}
	}
	return rs.polynomes[degree]
}

// Encode ..
func (rs *ReedSolomonEncoder) Encode(data []int, eccCount int) []int {
	generator := rs.getPolynomial(eccCount)
	info := NewGFPoly(rs.gf, data)
	info = info.MultByMonominal(eccCount, 1)
	_, remainder := info.Divide(generator)

	result := make([]int, eccCount)
	numZero := int(eccCount) - len(remainder.Coefficients)
	copy(result[numZero:], remainder.Coefficients)
	return result
}

// RuneToInt converts a rune between '0' and '9' to an integer between 0 and 9
// If the rune is outside of this range -1 is returned.
func RuneToInt(r rune) int {
	if r >= '0' && r <= '9' {
		return int(r - '0')
	}
	return -1
}

// IntToRune converts a digit 0 - 9 to the rune '0' - '9'. If the given int is outside
// of this range 'F' is returned!
func IntToRune(i int) rune {
	if i >= 0 && i <= 9 {
		return rune(i + '0')
	}
	return 'F'
}

// const
const (
	// TypeAztec ...
	TypeAztec           = "Aztec"
	TypeCodabar         = "Codabar"
	TypeCode128         = "Code 128"
	TypeCode39          = "Code 39"
	TypeCode93          = "Code 93"
	TypeDataMatrix      = "DataMatrix"
	TypeEAN8            = "EAN 8"
	TypeEAN13           = "EAN 13"
	TypePDF             = "PDF417"
	TypeQR              = "QR Code"
	Type2of5            = "2 of 5"
	Type2of5Interleaved = "2 of 5 (interleaved)"
)

// Metadata ... Contains some meta information about a barcode
type Metadata struct {
	// the name of the barcode kind
	CodeKind string
	// contains 1 for 1D barcodes or 2 for 2D barcodes
	Dimensions byte
}

// Barcode ... a rendered and encoded barcode
type Barcode interface {
	image.Image
	// returns some meta information about the barcode
	Metadata() Metadata
	// the data that was encoded in this barcode
	Content() string
}

// BarcodeIntCS ... Additional interface that some barcodes might implement to provide
// the value of its checksum.
type BarcodeIntCS interface {
	Barcode
	CheckSum() int
}

type wrapFunc func(x, y int) color.Color

type scaledBarcode struct {
	wrapped     Barcode
	wrapperFunc wrapFunc
	rect        image.Rectangle
}

type intCSscaledBC struct {
	scaledBarcode
}

func (bc *scaledBarcode) Content() string {
	return bc.wrapped.Content()
}

func (bc *scaledBarcode) Metadata() Metadata {
	return bc.wrapped.Metadata()
}

func (bc *scaledBarcode) ColorModel() color.Model {
	return bc.wrapped.ColorModel()
}

func (bc *scaledBarcode) Bounds() image.Rectangle {
	return bc.rect
}

func (bc *scaledBarcode) At(x, y int) color.Color {
	return bc.wrapperFunc(x, y)
}

func (bc *intCSscaledBC) CheckSum() int {
	if cs, ok := bc.wrapped.(BarcodeIntCS); ok {
		return cs.CheckSum()
	}
	return 0
}

// Scale returns a resized barcode with the given width and height.
func Scale(bc Barcode, width, height int) (Barcode, error) {
	switch bc.Metadata().Dimensions {
	case 1:
		return scale1DCode(bc, width, height)
	case 2:
		return scale2DCode(bc, width, height)
	}

	return nil, errors.New("unsupported barcode format")
}

func newScaledBC(wrapped Barcode, wrapperFunc wrapFunc, rect image.Rectangle) Barcode {
	result := &scaledBarcode{
		wrapped:     wrapped,
		wrapperFunc: wrapperFunc,
		rect:        rect,
	}

	if _, ok := wrapped.(BarcodeIntCS); ok {
		return &intCSscaledBC{*result}
	}
	return result
}

func scale2DCode(bc Barcode, width, height int) (Barcode, error) {
	orgBounds := bc.Bounds()
	orgWidth := orgBounds.Max.X - orgBounds.Min.X
	orgHeight := orgBounds.Max.Y - orgBounds.Min.Y

	factor := int(math.Min(float64(width)/float64(orgWidth), float64(height)/float64(orgHeight)))
	if factor <= 0 {
		return nil, fmt.Errorf("can not scale barcode to an image smaller than %dx%d", orgWidth, orgHeight)
	}

	offsetX := (width - (orgWidth * factor)) / 2
	offsetY := (height - (orgHeight * factor)) / 2

	wrap := func(x, y int) color.Color {
		if x < offsetX || y < offsetY {
			return color.White
		}
		x = (x - offsetX) / factor
		y = (y - offsetY) / factor
		if x >= orgWidth || y >= orgHeight {
			return color.White
		}
		return bc.At(x, y)
	}

	return newScaledBC(
		bc,
		wrap,
		image.Rect(0, 0, width, height),
	), nil
}

func scale1DCode(bc Barcode, width, height int) (Barcode, error) {
	orgBounds := bc.Bounds()
	orgWidth := orgBounds.Max.X - orgBounds.Min.X
	factor := int(float64(width) / float64(orgWidth))

	if factor <= 0 {
		return nil, fmt.Errorf("can not scale barcode to an image smaller than %dx1", orgWidth)
	}
	offsetX := (width - (orgWidth * factor)) / 2

	wrap := func(x, y int) color.Color {
		if x < offsetX {
			return color.White
		}
		x = (x - offsetX) / factor

		if x >= orgWidth {
			return color.White
		}
		return bc.At(x, 0)
	}

	return newScaledBC(
		bc,
		wrap,
		image.Rect(0, 0, width, height),
	), nil
}

// ### SVGO

// SVG defines the location of the generated SVG
type SVG struct {
	Writer io.Writer
}

// Offcolor defines the offset and color for gradients
type Offcolor struct {
	Offset  uint8
	Color   string
	Opacity float64
}

// Filterspec defines the specification of SVG filters
type Filterspec struct {
	In, In2, Result string
}

const (
	svgtop = `<?xml version="1.0"?>
<!-- Generated by SVGo -->
<svg`
	svginitfmt = `%s width="%d%s" height="%d%s"`
	svgns      = `
     xmlns="http://www.w3.org/2000/svg"
     xmlns:xlink="http://www.w3.org/1999/xlink">`
	vbfmt = `viewBox="%d %d %d %d"`

	emptyclose = "/>\n"
)

// New is the SVG constructor, specifying the io.Writer where the generated SVG is written.
func New(w io.Writer) *SVG { return &SVG{w} }

func (svg *SVG) print(a ...any) (n int, errno error) {
	return fmt.Fprint(svg.Writer, a...)
}

func (svg *SVG) println(a ...any) (n int, errno error) {
	return fmt.Fprintln(svg.Writer, a...)
}

func (svg *SVG) printf(format string, a ...any) (n int, errno error) {
	return fmt.Fprintf(svg.Writer, format, a...)
}

func (svg *SVG) genattr(ns []string) {
	for _, v := range ns {
		svg.printf("\n     %s", v)
	}
	svg.println(svgns)
}

// Structure, Metadata, Scripting, Style, Transformation, and Links

// Start begins the SVG document with the width w and height h.
// Other attributes may be optionally added, for example viewbox or additional namespaces
// Standard Reference: http://www.w3.org/TR/SVG11/struct.html#SVGElement
func (svg *SVG) Start(w, h int, ns ...string) {
	svg.printf(svginitfmt, svgtop, w, "", h, "")
	svg.genattr(ns)
}

// Startunit begins the SVG document, with width and height in the specified units
// Other attributes may be optionally added, for example viewbox or additional namespaces
func (svg *SVG) Startunit(w, h int, unit string, ns ...string) {
	svg.printf(svginitfmt, svgtop, w, unit, h, unit)
	svg.genattr(ns)
}

// Startpercent begins the SVG document, with width and height as percentages
// Other attributes may be optionally added, for example viewbox or additional namespaces
func (svg *SVG) Startpercent(w, h int, ns ...string) {
	svg.printf(svginitfmt, svgtop, w, "%", h, "%")
	svg.genattr(ns)
}

// Startview begins the SVG document, with the specified width, height, and viewbox
// Other attributes may be optionally added, for example viewbox or additional namespaces
func (svg *SVG) Startview(w, h, minx, miny, vw, vh int) {
	svg.Start(w, h, fmt.Sprintf(vbfmt, minx, miny, vw, vh))
}

// StartviewUnit begins the SVG document with the specified width, height, and unit
func (svg *SVG) StartviewUnit(w, h int, unit string, minx, miny, vw, vh int) {
	svg.Startunit(w, h, unit, fmt.Sprintf(vbfmt, minx, miny, vw, vh))
}

// Startraw begins the SVG document, passing arbitrary attributes
func (svg *SVG) Startraw(ns ...string) {
	svg.printf(svgtop)
	svg.genattr(ns)
}

// End the SVG document
func (svg *SVG) End() { svg.println("</svg>") }

// linkembed defines an element with a specified type,
// (for example "application/javascript", or "text/css").
// if the first variadic argument is a link, use only the link reference.
// Otherwise, treat those arguments as the text of the script (marked up as CDATA).
// if no data is specified, just close the element
func (svg *SVG) linkembed(tag, scriptype string, data ...string) {
	svg.printf(`<%s type="%s"`, tag, scriptype)
	switch {
	case len(data) == 1 && islink(data[0]):
		svg.printf(" %s/>\n", href(data[0]))

	case len(data) > 0:
		svg.printf(">\n<![CDATA[\n")
		for _, v := range data {
			svg.println(v)
		}
		svg.printf("]]>\n</%s>\n", tag)

	default:
		svg.println(`/>`)
	}
}

// Script defines a script with a specified type, (for example "application/javascript").
func (svg *SVG) Script(scriptype string, data ...string) {
	svg.linkembed("script", scriptype, data...)
}

// Style defines the specified style (for example "text/css")
func (svg *SVG) Style(scriptype string, data ...string) {
	svg.linkembed("style", scriptype, data...)
}

// Gstyle begins a group, with the specified style.
// Standard Reference: http://www.w3.org/TR/SVG11/struct.html#GElement
func (svg *SVG) Gstyle(s string) { svg.println(group("style", s)) }

// Gtransform begins a group, with the specified transform
// Standard Reference: http://www.w3.org/TR/SVG11/coords.html#TransformAttribute
func (svg *SVG) Gtransform(s string) { svg.println(group("transform", s)) }

// Translate begins coordinate translation, end with Gend()
// Standard Reference: http://www.w3.org/TR/SVG11/coords.html#TransformAttribute
func (svg *SVG) Translate(x, y int) { svg.Gtransform(translate(x, y)) }

// Scale scales the coordinate system by n, end with Gend()
// Standard Reference: http://www.w3.org/TR/SVG11/coords.html#TransformAttribute
func (svg *SVG) Scale(n float64) { svg.Gtransform(scale(n)) }

// ScaleXY scales the coordinate system by dx and dy, end with Gend()
// Standard Reference: http://www.w3.org/TR/SVG11/coords.html#TransformAttribute
func (svg *SVG) ScaleXY(dx, dy float64) { svg.Gtransform(scaleXY(dx, dy)) }

// SkewX skews the x coordinate system by angle a, end with Gend()
// Standard Reference: http://www.w3.org/TR/SVG11/coords.html#TransformAttribute
func (svg *SVG) SkewX(a float64) { svg.Gtransform(skewX(a)) }

// SkewY skews the y coordinate system by angle a, end with Gend()
// Standard Reference: http://www.w3.org/TR/SVG11/coords.html#TransformAttribute
func (svg *SVG) SkewY(a float64) { svg.Gtransform(skewY(a)) }

// SkewXY skews x and y coordinates by ax, ay respectively, end with Gend()
// Standard Reference: http://www.w3.org/TR/SVG11/coords.html#TransformAttribute
func (svg *SVG) SkewXY(ax, ay float64) { svg.Gtransform(skewX(ax) + " " + skewY(ay)) }

// Rotate rotates the coordinate system by r degrees, end with Gend()
// Standard Reference: http://www.w3.org/TR/SVG11/coords.html#TransformAttribute
func (svg *SVG) Rotate(r float64) { svg.Gtransform(rotate(r)) }

// TranslateRotate translates the coordinate system to (x,y), then rotates to r degrees, end with Gend()
func (svg *SVG) TranslateRotate(x, y int, r float64) {
	svg.Gtransform(translate(x, y) + " " + rotate(r))
}

// RotateTranslate rotates the coordinate system r degrees, then translates to (x,y), end with Gend()
func (svg *SVG) RotateTranslate(x, y int, r float64) {
	svg.Gtransform(rotate(r) + " " + translate(x, y))
}

// Group begins a group with arbitrary attributes
func (svg *SVG) Group(s ...string) { svg.printf("<g %s\n", endstyle(s, `>`)) }

// Gid begins a group, with the specified id
func (svg *SVG) Gid(s string) {
	svg.print(`<g id="`)
	xml.Escape(svg.Writer, []byte(s))
	svg.println(`">`)
}

// Gend ends a group (must be paired with Gsttyle, Gtransform, Gid).
func (svg *SVG) Gend() { svg.println(`</g>`) }

// ClipPath defines a clip path
func (svg *SVG) ClipPath(s ...string) { svg.printf(`<clipPath %s`, endstyle(s, `>`)) }

// ClipEnd ends a ClipPath
func (svg *SVG) ClipEnd() {
	svg.println(`</clipPath>`)
}

// Def begins a definition block.
// Standard Reference: http://www.w3.org/TR/SVG11/struct.html#DefsElement
func (svg *SVG) Def() { svg.println(`<defs>`) }

// DefEnd ends a definition block.
func (svg *SVG) DefEnd() { svg.println(`</defs>`) }

// Marker defines a marker
// Standard reference: http://www.w3.org/TR/SVG11/painting.html#MarkerElement
func (svg *SVG) Marker(id string, x, y, width, height int, s ...string) {
	svg.printf(`<marker id="%s" refX="%d" refY="%d" markerWidth="%d" markerHeight="%d" %s`,
		id, x, y, width, height, endstyle(s, ">\n"))
}

// MarkerEnd ends a marker
func (svg *SVG) MarkerEnd() { svg.println(`</marker>`) }

// Pattern defines a pattern with the specified dimensions.
// The putype can be either "user" or "obj", which sets the patternUnits
// attribute to be either userSpaceOnUse or objectBoundingBox
// Standard reference: http://www.w3.org/TR/SVG11/pservers.html#Patterns
func (svg *SVG) Pattern(id string, x, y, width, height int, putype string, s ...string) {
	puattr := "userSpaceOnUse"
	if putype != "user" {
		puattr = "objectBoundingBox"
	}
	svg.printf(`<pattern id="%s" x="%d" y="%d" width="%d" height="%d" patternUnits="%s" %s`,
		id, x, y, width, height, puattr, endstyle(s, ">\n"))
}

// PatternEnd ends a marker
func (svg *SVG) PatternEnd() { svg.println(`</pattern>`) }

// Desc specified the text of the description tag.
// Standard Reference: http://www.w3.org/TR/SVG11/struct.html#DescElement
func (svg *SVG) Desc(s string) { svg.tt("desc", s) }

// Title specified the text of the title tag.
// Standard Reference: http://www.w3.org/TR/SVG11/struct.html#TitleElement
func (svg *SVG) Title(s string) { svg.tt("title", s) }

// Link begins a link named "name", with the specified title.
// Standard Reference: http://www.w3.org/TR/SVG11/linking.html#Links
func (svg *SVG) Link(href, title string) {
	svg.printf("<a xlink:href=\"%s\" xlink:title=\"", href)
	xml.Escape(svg.Writer, []byte(title))
	svg.println("\">")
}

// LinkEnd ends a link.
func (svg *SVG) LinkEnd() { svg.println(`</a>`) }

// Use places the object referenced at link at the location x, y, with optional style.
// Standard Reference: http://www.w3.org/TR/SVG11/struct.html#UseElement
func (svg *SVG) Use(x, y int, link string, s ...string) {
	svg.printf(`<use %s %s %s`, loc(x, y), href(link), endstyle(s, emptyclose))
}

// Mask creates a mask with a specified id, dimension, and optional style.
func (svg *SVG) Mask(id string, x, y, w, h int, s ...string) {
	svg.printf(`<mask id="%s" x="%d" y="%d" width="%d" height="%d" %s`, id, x, y, w, h, endstyle(s, `>`))
}

// MaskEnd ends a Mask.
func (svg *SVG) MaskEnd() { svg.println(`</mask>`) }

// Shapes

// Circle centered at x,y, with radius r, with optional style.
// Standard Reference: http://www.w3.org/TR/SVG11/shapes.html#CircleElement
func (svg *SVG) Circle(x, y, r int, s ...string) {
	svg.printf(`<circle cx="%d" cy="%d" r="%d" %s`, x, y, r, endstyle(s, emptyclose))
}

// Ellipse centered at x,y, centered at x,y with radii w, and h, with optional style.
// Standard Reference: http://www.w3.org/TR/SVG11/shapes.html#EllipseElement
func (svg *SVG) Ellipse(x, y, w, h int, s ...string) {
	svg.printf(`<ellipse cx="%d" cy="%d" rx="%d" ry="%d" %s`,
		x, y, w, h, endstyle(s, emptyclose))
}

// Polygon draws a series of line segments using an array of x, y coordinates, with optional style.
// Standard Reference: http://www.w3.org/TR/SVG11/shapes.html#PolygonElement
func (svg *SVG) Polygon(x, y []int, s ...string) {
	svg.poly(x, y, "polygon", s...)
}

// Rect draws a rectangle with upper left-hand corner at x,y, with width w, and height h, with optional style
// Standard Reference: http://www.w3.org/TR/SVG11/shapes.html#RectElement
func (svg *SVG) Rect(x, y, w, h int, s ...string) {
	svg.printf(`<rect %s %s`, dim(x, y, w, h), endstyle(s, emptyclose))
}

// CenterRect draws a rectangle with its center at x,y, with width w, and height h, with optional style
func (svg *SVG) CenterRect(x, y, w, h int, s ...string) {
	svg.Rect(x-(w/2), y-(h/2), w, h, s...)
}

// Roundrect draws a rounded rectangle with upper the left-hand corner at x,y,
// with width w, and height h. The radii for the rounded portion
// are specified by rx (width), and ry (height).
// Style is optional.
// Standard Reference: http://www.w3.org/TR/SVG11/shapes.html#RectElement
func (svg *SVG) Roundrect(x, y, w, h, rx, ry int, s ...string) {
	svg.printf(`<rect %s rx="%d" ry="%d" %s`, dim(x, y, w, h), rx, ry, endstyle(s, emptyclose))
}

// Square draws a square with upper left corner at x,y with sides of length l, with optional style.
func (svg *SVG) Square(x, y, l int, s ...string) {
	svg.Rect(x, y, l, l, s...)
}

// Paths

// Path draws an arbitrary path, the caller is responsible for structuring the path data
func (svg *SVG) Path(d string, s ...string) {
	svg.printf(`<path d="%s" %s`, d, endstyle(s, emptyclose))
}

// Arc draws an elliptical arc, with optional style, beginning coordinate at sx,sy, ending coordinate at ex, ey
// width and height of the arc are specified by ax, ay, the x axis rotation is r
// if sweep is true, then the arc will be drawn in a "positive-angle" direction (clockwise), if false,
// the arc is drawn counterclockwise.
// if large is true, the arc sweep angle is greater than or equal to 180 degrees,
// otherwise the arc sweep is less than 180 degrees
// http://www.w3.org/TR/SVG11/paths.html#PathDataEllipticalArcCommands
func (svg *SVG) Arc(sx, sy, ax, ay, r int, large, sweep bool, ex, ey int, s ...string) {
	svg.printf(`%s A%s %d %s %s %s" %s`,
		ptag(sx, sy), coord(ax, ay), r, onezero(large), onezero(sweep), coord(ex, ey), endstyle(s, emptyclose))
}

// Bezier draws a cubic bezier curve, with optional style, beginning at sx,sy, ending at ex,ey
// with control points at cx,cy and px,py.
// Standard Reference: http://www.w3.org/TR/SVG11/paths.html#PathDataCubicBezierCommands
func (svg *SVG) Bezier(sx, sy, cx, cy, px, py, ex, ey int, s ...string) {
	svg.printf(`%s C%s %s %s" %s`,
		ptag(sx, sy), coord(cx, cy), coord(px, py), coord(ex, ey), endstyle(s, emptyclose))
}

// Qbez draws a quadratic bezier curver, with optional style
// beginning at sx,sy, ending at ex, sy with control points at cx, cy
// Standard Reference: http://www.w3.org/TR/SVG11/paths.html#PathDataQuadraticBezierCommands
func (svg *SVG) Qbez(sx, sy, cx, cy, ex, ey int, s ...string) {
	svg.printf(`%s Q%s %s" %s`,
		ptag(sx, sy), coord(cx, cy), coord(ex, ey), endstyle(s, emptyclose))
}

// Qbezier draws a Quadratic Bezier curve, with optional style, beginning at sx, sy, ending at tx,ty
// with control points are at cx,cy, ex,ey.
// Standard Reference: http://www.w3.org/TR/SVG11/paths.html#PathDataQuadraticBezierCommands
func (svg *SVG) Qbezier(sx, sy, cx, cy, ex, ey, tx, ty int, s ...string) {
	svg.printf(`%s Q%s %s T%s" %s`,
		ptag(sx, sy), coord(cx, cy), coord(ex, ey), coord(tx, ty), endstyle(s, emptyclose))
}

// Lines

// Line draws a straight line between two points, with optional style.
// Standard Reference: http://www.w3.org/TR/SVG11/shapes.html#LineElement
func (svg *SVG) Line(x1, y1, x2, y2 int, s ...string) {
	svg.printf(`<line x1="%d" y1="%d" x2="%d" y2="%d" %s`, x1, y1, x2, y2, endstyle(s, emptyclose))
}

// Polyline draws connected lines between coordinates, with optional style.
// Standard Reference: http://www.w3.org/TR/SVG11/shapes.html#PolylineElement
func (svg *SVG) Polyline(x, y []int, s ...string) {
	svg.poly(x, y, "polyline", s...)
}

// Image places at x,y (upper left hand corner), the image with
// width w, and height h, referenced at link, with optional style.
// Standard Reference: http://www.w3.org/TR/SVG11/struct.html#ImageElement
func (svg *SVG) Image(x, y, w, h int, link string, s ...string) {
	svg.printf(`<image %s %s %s`, dim(x, y, w, h), href(link), endstyle(s, emptyclose))
}

// Text places the specified text, t at x,y according to the style specified in s
// Standard Reference: http://www.w3.org/TR/SVG11/text.html#TextElement
func (svg *SVG) Text(x, y int, t string, s ...string) {
	svg.printf(`<text %s %s`, loc(x, y), endstyle(s, ">"))
	xml.Escape(svg.Writer, []byte(t))
	svg.println(`</text>`)
}

// Textspan begins text, assuming a tspan will be included, end with TextEnd()
// Standard Reference: https://www.w3.org/TR/SVG11/text.html#TSpanElement
func (svg *SVG) Textspan(x, y int, t string, s ...string) {
	svg.printf(`<text %s %s`, loc(x, y), endstyle(s, ">"))
	xml.Escape(svg.Writer, []byte(t))
}

// Span makes styled spanned text, should be proceeded by Textspan
// Standard Reference: https://www.w3.org/TR/SVG11/text.html#TSpanElement
func (svg *SVG) Span(t string, s ...string) {
	if len(s) == 0 {
		xml.Escape(svg.Writer, []byte(t))
		return
	}
	svg.printf(`<tspan %s`, endstyle(s, ">"))
	xml.Escape(svg.Writer, []byte(t))
	svg.printf(`</tspan>`)
}

// TextEnd ends spanned text
// Standard Reference: https://www.w3.org/TR/SVG11/text.html#TSpanElement
func (svg *SVG) TextEnd() {
	svg.println(`</text>`)
}

// Textpath places text optionally styled text along a previously defined path
// Standard Reference: http://www.w3.org/TR/SVG11/text.html#TextPathElement
func (svg *SVG) Textpath(t, pathid string, s ...string) {
	svg.printf("<text %s<textPath xlink:href=\"%s\">", endstyle(s, ">"), pathid)
	xml.Escape(svg.Writer, []byte(t))
	svg.println(`</textPath></text>`)
}

// Textlines places a series of lines of text starting at x,y, at the specified size, fill, and alignment.
// Each line is spaced according to the spacing argument
func (svg *SVG) Textlines(x, y int, s []string, size, spacing int, fill, align string) {
	svg.Gstyle(fmt.Sprintf("font-size:%dpx;fill:%s;text-anchor:%s", size, fill, align))
	for _, t := range s {
		svg.Text(x, y, t)
		y += spacing
	}
	svg.Gend()
}

// Colors

// RGB specifies a fill color in terms of a (r)ed, (g)reen, (b)lue triple.
// Standard reference: http://www.w3.org/TR/css3-color/
func (svg *SVG) RGB(r, g, b int) string {
	return fmt.Sprintf(`fill:rgb(%d,%d,%d)`, r, g, b)
}

// RGBA specifies a fill color in terms of a (r)ed, (g)reen, (b)lue triple and opacity.
func (svg *SVG) RGBA(r, g, b int, a float64) string {
	return fmt.Sprintf(`fill-opacity:%.2f; %s`, a, svg.RGB(r, g, b))
}

// Gradients

// LinearGradient constructs a linear color gradient identified by id,
// along the vector defined by (x1,y1), and (x2,y2).
// The stop color sequence defined in sc. Coordinates are expressed as percentages.
func (svg *SVG) LinearGradient(id string, x1, y1, x2, y2 uint8, sc []Offcolor) {
	svg.printf("<linearGradient id=\"%s\" x1=\"%d%%\" y1=\"%d%%\" x2=\"%d%%\" y2=\"%d%%\">\n",
		id, pct(x1), pct(y1), pct(x2), pct(y2))
	svg.stopcolor(sc)
	svg.println("</linearGradient>")
}

// RadialGradient constructs a radial color gradient identified by id,
// centered at (cx,cy), with a radius of r.
// (fx, fy) define the location of the focal point of the light source.
// The stop color sequence defined in sc.
// Coordinates are expressed as percentages.
func (svg *SVG) RadialGradient(id string, cx, cy, r, fx, fy uint8, sc []Offcolor) {
	svg.printf("<radialGradient id=\"%s\" cx=\"%d%%\" cy=\"%d%%\" r=\"%d%%\" fx=\"%d%%\" fy=\"%d%%\">\n",
		id, pct(cx), pct(cy), pct(r), pct(fx), pct(fy))
	svg.stopcolor(sc)
	svg.println("</radialGradient>")
}

// stopcolor is a utility function used by the gradient functions
// to define a sequence of offsets (expressed as percentages) and colors
func (svg *SVG) stopcolor(oc []Offcolor) {
	for _, v := range oc {
		svg.printf("<stop offset=\"%d%%\" stop-color=\"%s\" stop-opacity=\"%.2f\"/>\n",
			pct(v.Offset), v.Color, v.Opacity)
	}
}

// Filter Effects:
// Most functions have common attributes (in, in2, result) defined in type Filterspec
// used as a common first argument.

// Filter begins a filter set
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#FilterElement
func (svg *SVG) Filter(id string, s ...string) {
	svg.printf(`<filter id="%s" %s`, id, endstyle(s, ">\n"))
}

// Fend ends a filter set
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#FilterElement
func (svg *SVG) Fend() {
	svg.println(`</filter>`)
}

// FeBlend specifies a Blend filter primitive
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feBlendElement
func (svg *SVG) FeBlend(fs Filterspec, mode string, s ...string) {
	switch mode {
	case "normal", "multiply", "screen", "darken", "lighten":
	default:
		mode = "normal"
	}
	svg.printf(`<feBlend %s mode="%s" %s`,
		fsattr(fs), mode, endstyle(s, emptyclose))
}

// FeColorMatrix specifies a color matrix filter primitive, with matrix values
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feColorMatrixElement
func (svg *SVG) FeColorMatrix(fs Filterspec, values [20]float64, s ...string) {
	svg.printf(`<feColorMatrix %s type="matrix" values="`, fsattr(fs))
	for _, v := range values {
		svg.printf(`%g `, v)
	}
	svg.printf(`" %s`, endstyle(s, emptyclose))
}

// FeColorMatrixHue specifies a color matrix filter primitive, with hue rotation values
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feColorMatrixElement
func (svg *SVG) FeColorMatrixHue(fs Filterspec, value float64, s ...string) {
	if value < -360 || value > 360 {
		value = 0
	}
	svg.printf(`<feColorMatrix %s type="hueRotate" values="%g" %s`,
		fsattr(fs), value, endstyle(s, emptyclose))
}

// FeColorMatrixSaturate specifies a color matrix filter primitive, with saturation values
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feColorMatrixElement
func (svg *SVG) FeColorMatrixSaturate(fs Filterspec, value float64, s ...string) {
	if value < 0 || value > 1 {
		value = 1
	}
	svg.printf(`<feColorMatrix %s type="saturate" values="%g" %s`,
		fsattr(fs), value, endstyle(s, emptyclose))
}

// FeColorMatrixLuminence specifies a color matrix filter primitive, with luminence values
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feColorMatrixElement
func (svg *SVG) FeColorMatrixLuminence(fs Filterspec, s ...string) {
	svg.printf(`<feColorMatrix %s type="luminenceToAlpha" %s`,
		fsattr(fs), endstyle(s, emptyclose))
}

// FeComponentTransfer begins a feComponent filter element
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feComponentTransferElement
func (svg *SVG) FeComponentTransfer() {
	svg.println(`<feComponentTransfer>`)
}

// FeCompEnd ends a feComponent filter element
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feComponentTransferElement
func (svg *SVG) FeCompEnd() {
	svg.println(`</feComponentTransfer>`)
}

// FeComposite specifies a feComposite filter primitive
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feCompositeElement
func (svg *SVG) FeComposite(fs Filterspec, operator string, k1, k2, k3, k4 int, s ...string) {
	switch operator {
	case "over", "in", "out", "atop", "xor", "arithmetic":
	default:
		operator = "over"
	}
	svg.printf(`<feComposite %s operator="%s" k1="%d" k2="%d" k3="%d" k4="%d" %s`,
		fsattr(fs), operator, k1, k2, k3, k4, endstyle(s, emptyclose))
}

// FeConvolveMatrix specifies a feConvolveMatrix filter primitive
// Standard referencd: http://www.w3.org/TR/SVG11/filters.html#feConvolveMatrixElement
func (svg *SVG) FeConvolveMatrix(fs Filterspec, matrix [9]int, s ...string) {
	svg.printf(`<feConvolveMatrix %s kernelMatrix="%d %d %d %d %d %d %d %d %d" %s`,
		fsattr(fs),
		matrix[0], matrix[1], matrix[2],
		matrix[3], matrix[4], matrix[5],
		matrix[6], matrix[7], matrix[8], endstyle(s, emptyclose))
}

// FeDiffuseLighting specifies a diffuse lighting filter primitive,
// a container for light source elements, end with DiffuseEnd()
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feComponentTransferElement
func (svg *SVG) FeDiffuseLighting(fs Filterspec, scale, constant float64, s ...string) {
	svg.printf(`<feDiffuseLighting %s surfaceScale="%g" diffuseConstant="%g" %s`,
		fsattr(fs), scale, constant, endstyle(s, `>`))
}

// FeDiffEnd ends a diffuse lighting filter primitive container
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feDiffuseLightingElement
func (svg *SVG) FeDiffEnd() {
	svg.println(`</feDiffuseLighting>`)
}

// FeDisplacementMap specifies a feDisplacementMap filter primitive
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feDisplacementMapElement
func (svg *SVG) FeDisplacementMap(fs Filterspec, scale float64, xchannel, ychannel string, s ...string) {
	svg.printf(`<feDisplacementMap %s scale="%g" xChannelSelector="%s" yChannelSelector="%s" %s`,
		fsattr(fs), scale, imgchannel(xchannel), ychannel, endstyle(s, emptyclose))
}

// FeDistantLight specifies a feDistantLight filter primitive
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feDistantLightElement
func (svg *SVG) FeDistantLight(fs Filterspec, azimuth, elevation float64, s ...string) {
	svg.printf(`<feDistantLight %s azimuth="%g" elevation="%g" %s`,
		fsattr(fs), azimuth, elevation, endstyle(s, emptyclose))
}

// FeFlood specifies a flood filter primitive
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feFloodElement
func (svg *SVG) FeFlood(fs Filterspec, co string, opacity float64, s ...string) {
	svg.printf(`<feFlood %s flood-color="%s" flood-opacity="%g" %s`,
		fsattr(fs), co, opacity, endstyle(s, emptyclose))
}

// FeFunc{linear|Gamma|Table|Discrete} specify various types of feFunc{R|G|B|A} filter primitives
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feComponentTransferElement

// FeFuncLinear specifies a linear style function for the feFunc{R|G|B|A} filter element
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feComponentTransferElement
func (svg *SVG) FeFuncLinear(channel string, slope, intercept float64) {
	svg.printf(`<feFunc%s type="linear" slope="%g" intercept="%g"%s`,
		imgchannel(channel), slope, intercept, emptyclose)
}

// FeFuncGamma specifies the curve values for gamma correction for the feFunc{R|G|B|A} filter element
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feComponentTransferElement
func (svg *SVG) FeFuncGamma(channel string, amplitude, exponent, offset float64) {
	svg.printf(`<feFunc%s type="gamma" amplitude="%g" exponent="%g" offset="%g"%s`,
		imgchannel(channel), amplitude, exponent, offset, emptyclose)
}

// FeFuncTable specifies the table of values for the feFunc{R|G|B|A} filter element
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feComponentTransferElement
func (svg *SVG) FeFuncTable(channel string, tv []float64) {
	svg.printf(`<feFunc%s type="table"`, imgchannel(channel))
	svg.tablevalues(`tableValues`, tv)
}

// FeFuncDiscrete specifies the discrete values for the feFunc{R|G|B|A} filter element
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feComponentTransferElement
func (svg *SVG) FeFuncDiscrete(channel string, tv []float64) {
	svg.printf(`<feFunc%s type="discrete"`, imgchannel(channel))
	svg.tablevalues(`tableValues`, tv)
}

// FeGaussianBlur specifies a Gaussian Blur filter primitive
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feGaussianBlurElement
func (svg *SVG) FeGaussianBlur(fs Filterspec, stdx, stdy float64, s ...string) {
	if stdx < 0 {
		stdx = 0
	}
	if stdy < 0 {
		stdy = 0
	}
	svg.printf(`<feGaussianBlur %s stdDeviation="%g %g" %s`,
		fsattr(fs), stdx, stdy, endstyle(s, emptyclose))
}

// FeImage specifies a feImage filter primitive
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feImageElement
func (svg *SVG) FeImage(href, result string, s ...string) {
	svg.printf(`<feImage xlink:href="%s" result="%s" %s`,
		href, result, endstyle(s, emptyclose))
}

// FeMerge specifies a feMerge filter primitive, containing feMerge elements
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feMergeElement
func (svg *SVG) FeMerge(nodes []string, _ ...string) {
	svg.println(`<feMerge>`)
	for _, n := range nodes {
		svg.printf("<feMergeNode in=\"%s\"/>\n", n)
	}
	svg.println(`</feMerge>`)
}

// FeMorphology specifies a feMorphologyLight filter primitive
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feMorphologyElement
func (svg *SVG) FeMorphology(fs Filterspec, operator string, xradius, yradius float64, s ...string) {
	switch operator {
	case "erode", "dilate":
	default:
		operator = "erode"
	}
	svg.printf(`<feMorphology %s operator="%s" radius="%g %g" %s`,
		fsattr(fs), operator, xradius, yradius, endstyle(s, emptyclose))
}

// FeOffset specifies the feOffset filter primitive
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feOffsetElement
func (svg *SVG) FeOffset(fs Filterspec, dx, dy int, s ...string) {
	svg.printf(`<feOffset %s dx="%d" dy="%d" %s`,
		fsattr(fs), dx, dy, endstyle(s, emptyclose))
}

// FePointLight specifies a fePpointLight filter primitive
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#fePointLightElement
func (svg *SVG) FePointLight(x, y, z float64, s ...string) {
	svg.printf(`<fePointLight x="%g" y="%g" z="%g" %s`,
		x, y, z, endstyle(s, emptyclose))
}

// FeSpecularLighting specifies a specular lighting filter primitive,
// a container for light source elements, end with SpecularEnd()
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feSpecularLightingElement
func (svg *SVG) FeSpecularLighting(fs Filterspec, scale, constant float64, exponent int, co string, s ...string) {
	svg.printf(`<feSpecularLighting %s surfaceScale="%g" specularConstant="%g" specularExponent="%d" lighting-color="%s" %s`,
		fsattr(fs), scale, constant, exponent, co, endstyle(s, ">\n"))
}

// FeSpecEnd ends a specular lighting filter primitive container
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feSpecularLightingElement
func (svg *SVG) FeSpecEnd() {
	svg.println(`</feSpecularLighting>`)
}

// FeSpotLight specifies a feSpotLight filter primitive
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feSpotLightElement
func (svg *SVG) FeSpotLight(fs Filterspec, x, y, z, px, py, pz float64, s ...string) {
	svg.printf(`<feSpotLight %s x="%g" y="%g" z="%g" pointsAtX="%g" pointsAtY="%g" pointsAtZ="%g" %s`,
		fsattr(fs), x, y, z, px, py, pz, endstyle(s, emptyclose))
}

// FeTile specifies the tile utility filter primitive
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feTileElement
func (svg *SVG) FeTile(fs Filterspec, _ string, s ...string) {
	svg.printf(`<feTile %s %s`, fsattr(fs), endstyle(s, emptyclose))
}

// FeTurbulence specifies a turbulence filter primitive
// Standard reference: http://www.w3.org/TR/SVG11/filters.html#feTurbulenceElement
func (svg *SVG) FeTurbulence(fs Filterspec, ftype string, bfx, bfy float64, octaves int, seed int64, stitch bool, s ...string) {
	if bfx < 0 || bfx > 1 {
		bfx = 0
	}
	if bfy < 0 || bfy > 1 {
		bfy = 0
	}
	switch ftype[0:1] {
	case "f", "F":
		ftype = "fractalNoise"
	case "t", "T":
		ftype = "turbulence"
	default:
		ftype = "turbulence"
	}

	var ss string
	if stitch {
		ss = "stitch"
	} else {
		ss = "noStitch"
	}
	svg.printf(`<feTurbulence %s type="%s" baseFrequency="%.2f %.2f" numOctaves="%d" seed="%d" stitchTiles="%s" %s`,
		fsattr(fs), ftype, bfx, bfy, octaves, seed, ss, endstyle(s, emptyclose))
}

// Filter Effects convenience functions, modeled after CSS versions

// Blur emulates the CSS blur filter
func (svg *SVG) Blur(p float64) {
	svg.FeGaussianBlur(Filterspec{}, p, p)
}

// Brightness emulates the CSS brightness filter
func (svg *SVG) Brightness(p float64) {
	svg.FeComponentTransfer()
	svg.FeFuncLinear("R", p, 0)
	svg.FeFuncLinear("G", p, 0)
	svg.FeFuncLinear("B", p, 0)
	svg.FeCompEnd()
}

// Contrast emulates the CSS contrast filter
//func (svg *SVG) Contrast(p float64) {
//}

// Dropshadow emulates the CSS dropshadow filter
//func (svg *SVG) Dropshadow(p float64) {
//}

// Grayscale eumulates the CSS grayscale filter
func (svg *SVG) Grayscale() {
	svg.FeColorMatrixSaturate(Filterspec{}, 0)
}

// HueRotate eumulates the CSS huerotate filter
func (svg *SVG) HueRotate(a float64) {
	svg.FeColorMatrixHue(Filterspec{}, a)
}

// Invert eumulates the CSS invert filter
func (svg *SVG) Invert() {
	svg.FeComponentTransfer()
	svg.FeFuncTable("R", []float64{1, 0})
	svg.FeFuncTable("G", []float64{1, 0})
	svg.FeFuncTable("B", []float64{1, 0})
	svg.FeCompEnd()
}

// Saturate eumulates the CSS saturate filter
func (svg *SVG) Saturate(p float64) {
	svg.FeColorMatrixSaturate(Filterspec{}, p)
}

// Sepia applies a sepia tone, emulating the CSS sepia filter
func (svg *SVG) Sepia() {
	sepiamatrix := [20]float64{
		0.280, 0.450, 0.05, 0, 0,
		0.140, 0.390, 0.04, 0, 0,
		0.080, 0.280, 0.03, 0, 0,
		0, 0, 0, 1, 0,
	}
	svg.FeColorMatrix(Filterspec{}, sepiamatrix)
}

// Animation

// Animate animates the specified link, using the specified attribute
// The animation starts at coordinate from, terminates at to, and repeats as specified
func (svg *SVG) Animate(link, attr string, from, to int, duration float64, repeat int, s ...string) {
	svg.printf(`<animate %s attributeName="%s" from="%d" to="%d" dur="%gs" repeatCount="%s" %s`,
		href(link), attr, from, to, duration, repeatString(repeat), endstyle(s, emptyclose))
}

// AnimateMotion animates the referenced object along the specified path
func (svg *SVG) AnimateMotion(link, path string, duration float64, repeat int, s ...string) {
	svg.printf(`<animateMotion %s dur="%gs" repeatCount="%s" %s<mpath %s/></animateMotion>
`, href(link), duration, repeatString(repeat), endstyle(s, ">"), href(path))
}

// AnimateTransform animates in the context of SVG transformations
func (svg *SVG) AnimateTransform(link, ttype, from, to string, duration float64, repeat int, s ...string) {
	svg.printf(`<animateTransform %s attributeName="transform" type="%s" from="%s" to="%s" dur="%gs" repeatCount="%s" %s`,
		href(link), ttype, from, to, duration, repeatString(repeat), endstyle(s, emptyclose))
}

// AnimateTranslate animates the translation transformation
func (svg *SVG) AnimateTranslate(link string, fx, fy, tx, ty int, duration float64, repeat int, s ...string) {
	svg.AnimateTransform(link, "translate", coordpair(fx, fy), coordpair(tx, ty), duration, repeat, s...)
}

// AnimateRotate animates the rotation transformation
func (svg *SVG) AnimateRotate(link string, fs, fc, fe, ts, tc, te int, duration float64, repeat int, s ...string) {
	svg.AnimateTransform(link, "rotate", sce(fs, fc, fe), sce(ts, tc, te), duration, repeat, s...)
}

// AnimateScale animates the scale transformation
func (svg *SVG) AnimateScale(link string, from, to, duration float64, repeat int, s ...string) {
	svg.AnimateTransform(link, "scale", fmt.Sprintf("%g", from), fmt.Sprintf("%g", to), duration, repeat, s...)
}

// AnimateSkewX animates the skewX transformation
func (svg *SVG) AnimateSkewX(link string, from, to, duration float64, repeat int, s ...string) {
	svg.AnimateTransform(link, "skewX", fmt.Sprintf("%g", from), fmt.Sprintf("%g", to), duration, repeat, s...)
}

// AnimateSkewY animates the skewY transformation
func (svg *SVG) AnimateSkewY(link string, from, to, duration float64, repeat int, s ...string) {
	svg.AnimateTransform(link, "skewY", fmt.Sprintf("%g", from), fmt.Sprintf("%g", to), duration, repeat, s...)
}

// Utility

// Grid draws a grid at the specified coordinate, dimensions, and spacing, with optional style.
func (svg *SVG) Grid(x, y, w, h, n int, s ...string) {
	if len(s) > 0 {
		svg.Gstyle(s[0])
	}
	for ix := x; ix <= x+w; ix += n {
		svg.Line(ix, y, ix, y+h)
	}

	for iy := y; iy <= y+h; iy += n {
		svg.Line(x, iy, x+w, iy)
	}
	if len(s) > 0 {
		svg.Gend()
	}
}

// Support functions

// coordpair returns a coordinate pair as a string
func coordpair(x, y int) string {
	return fmt.Sprintf("%d %d", x, y)
}

// sce makes start, center, end coordinates string for animate transformations
func sce(start, center, end int) string {
	return fmt.Sprintf("%d %d %d", start, center, end)
}

// repeatString computes the repeat string for animation methods
// repeat <= 0 --> "indefinite", otherwise the integer string
func repeatString(n int) string {
	if n > 0 {
		return fmt.Sprintf("%d", n)
	}
	return "indefinite"
}

// style returns a style name,attribute string
func style(s string) string {
	if len(s) > 0 {
		return fmt.Sprintf(`style="%s"`, s)
	}
	return s
}

// pp returns a series of polygon points
func (svg *SVG) pp(x, y []int, tag string) {
	svg.print(tag)
	if len(x) != len(y) {
		svg.print(" ")
		return
	}
	lx := len(x) - 1
	for i := 0; i < lx; i++ {
		svg.print(coord(x[i], y[i]) + " ")
	}
	svg.print(coord(x[lx], y[lx]))
}

// endstyle modifies an SVG object, with either a series of name="value" pairs,
// or a single string containing a style
func endstyle(s []string, endtag string) string {
	if len(s) > 0 {
		nv := ""
		for i := 0; i < len(s); i++ {
			if strings.Index(s[i], "=") > 0 {
				nv += (s[i]) + " "
			} else {
				nv += style(s[i]) + " "
			}
		}
		return nv + endtag
	}
	return endtag
}

// tt creates a xml element, tag containing s
func (svg *SVG) tt(tag, s string) {
	svg.print("<" + tag + ">")
	xml.Escape(svg.Writer, []byte(s))
	svg.println("</" + tag + ">")
}

// poly compiles the polygon element
func (svg *SVG) poly(x, y []int, tag string, s ...string) {
	svg.pp(x, y, "<"+tag+" points=\"")
	svg.print(`" ` + endstyle(s, "/>\n"))
}

// onezero returns "0" or "1"
func onezero(flag bool) string {
	if flag {
		return "1"
	}
	return "0"
}

// pct returns a percetage, capped at 100
func pct(n uint8) uint8 {
	if n > 100 {
		return 100
	}
	return n
}

// islink determines if a string is a script reference
func islink(link string) bool {
	return strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "#") ||
		strings.HasPrefix(link, "../") || strings.HasPrefix(link, "./")
}

// group returns a group element
func group(tag, value string) string { return fmt.Sprintf(`<g %s="%s">`, tag, value) }

// scale return the scale string for the transform
func scale(n float64) string { return fmt.Sprintf(`scale(%g)`, n) }

// scaleXY return the scale string for the transform
func scaleXY(dx, dy float64) string { return fmt.Sprintf(`scale(%g,%g)`, dx, dy) }

// skewx returns the skewX string for the transform
func skewX(angle float64) string { return fmt.Sprintf(`skewX(%g)`, angle) }

// skewx returns the skewX string for the transform
func skewY(angle float64) string { return fmt.Sprintf(`skewY(%g)`, angle) }

// rotate returns the rotate string for the transform
func rotate(r float64) string { return fmt.Sprintf(`rotate(%g)`, r) }

// translate returns the translate string for the transform
func translate(x, y int) string { return fmt.Sprintf(`translate(%d,%d)`, x, y) }

// coord returns a coordinate string
func coord(x, y int) string { return fmt.Sprintf(`%d,%d`, x, y) }

// ptag returns the beginning of the path element
func ptag(x, y int) string { return fmt.Sprintf(`<path d="M%s`, coord(x, y)) }

// loc returns the x and y coordinate attributes
func loc(x, y int) string { return fmt.Sprintf(`x="%d" y="%d"`, x, y) }

// href returns the href name and attribute
func href(s string) string { return fmt.Sprintf(`xlink:href="%s"`, s) }

// dim returns the dimension string (x, y coordinates and width, height)
func dim(x, y, w, h int) string {
	return fmt.Sprintf(`x="%d" y="%d" width="%d" height="%d"`, x, y, w, h)
}

// fsattr returns the XML attribute representation of a filterspec, ignoring empty attributes
func fsattr(s Filterspec) string {
	attrs := ""
	if len(s.In) > 0 {
		attrs += fmt.Sprintf(`in="%s" `, s.In)
	}
	if len(s.In2) > 0 {
		attrs += fmt.Sprintf(`in2="%s" `, s.In2)
	}
	if len(s.Result) > 0 {
		attrs += fmt.Sprintf(`result="%s" `, s.Result)
	}
	return attrs
}

// tablevalues outputs a series of values as a XML attribute
func (svg *SVG) tablevalues(s string, t []float64) {
	svg.printf(` %s="`, s)
	for i := 0; i < len(t)-1; i++ {
		svg.printf("%g ", t[i])
	}
	svg.printf(`%g"%s`, t[len(t)-1], emptyclose)
}

// imgchannel validates the image channel indicator
func imgchannel(c string) string {
	switch c {
	case "R", "G", "B", "A":
		return c
	case "r", "g", "b", "a":
		return strings.ToUpper(c)
	case "red", "green", "blue", "alpha":
		return strings.ToUpper(c[0:1])
	case "Red", "Green", "Blue", "Alpha":
		return c[0:1]
	}
	return "R"
}
