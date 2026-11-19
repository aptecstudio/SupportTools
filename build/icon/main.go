// SPDX-License-Identifier: GPL-3.0-or-later

// Il comando icon disegna l'icona di Support Tools e la salva in formato
// .ico con tutte le dimensioni usate da Windows.
//
// Uso, dalla radice del repository:
//
//	go run ./build/icon [-o cmd/supporttools/supporttools.ico] [-preview cartella]
//
// L'icona è descritta da forme geometriche (quadrato arrotondato e chiave
// inglese), quindi ogni dimensione viene disegnata alla sua risoluzione
// invece di essere ridotta da un'immagine grande.
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

// sizes sono le dimensioni incluse nell'icona: quelle standard di Windows
// per barra delle applicazioni, Esplora risorse e schermi ad alta densità.
var sizes = []int{16, 20, 24, 32, 40, 48, 64, 256}

// pngMinSize è la dimensione da cui le immagini vengono salvate come PNG
// (supportato da Windows Vista in poi); le più piccole restano bitmap.
const pngMinSize = 256

// supersampling è il numero di campioni per lato di ogni pixel, per
// ottenere bordi sfumati.
const supersampling = 8

func main() {
	out := flag.String("o", filepath.Join("cmd", "supporttools", "supporttools.ico"), "file .ico da creare")
	preview := flag.String("preview", "", "cartella in cui salvare un'anteprima PNG di ogni dimensione")
	flag.Parse()

	if err := run(*out, *preview); err != nil {
		fmt.Fprintln(os.Stderr, "icon:", err)
		os.Exit(1)
	}
}

func run(out, preview string) error {
	images := make([]*image.NRGBA, len(sizes))
	for i, s := range sizes {
		images[i] = render(s)
	}
	data, err := encodeICO(images)
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return err
	}
	fmt.Println("icona salvata in", out)

	if preview == "" {
		return nil
	}
	if err := os.MkdirAll(preview, 0o755); err != nil {
		return err
	}
	for i, img := range images {
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return err
		}
		name := filepath.Join(preview, fmt.Sprintf("icon-%d.png", sizes[i]))
		if err := os.WriteFile(name, buf.Bytes(), 0o644); err != nil {
			return err
		}
	}
	fmt.Println("anteprime salvate in", preview)
	return nil
}

// render disegna l'icona di lato size pixel.
func render(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	withHole := size >= 32
	const n = supersampling * supersampling
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			// Somma dei colori premoltiplicati per l'alfa dei campioni.
			var r, g, b, a float64
			for sy := 0; sy < supersampling; sy++ {
				for sx := 0; sx < supersampling; sx++ {
					x := (float64(px) + (float64(sx)+0.5)/supersampling) / float64(size)
					y := (float64(py) + (float64(sy)+0.5)/supersampling) / float64(size)
					c, ok := shade(x, y, withHole)
					if ok {
						r += c.r
						g += c.g
						b += c.b
						a++
					}
				}
			}
			if a == 0 {
				continue
			}
			img.SetNRGBA(px, py, color.NRGBA{
				R: to8(r / a),
				G: to8(g / a),
				B: to8(b / a),
				A: to8(a / n),
			})
		}
	}
	return img
}

// rgb è un colore con componenti da 0 a 1.
type rgb struct{ r, g, b float64 }

var (
	bgTop    = rgb{0x2F / 255.0, 0x86 / 255.0, 0xE8 / 255.0}
	bgBottom = rgb{0x17 / 255.0, 0x4A / 255.0, 0x9A / 255.0}
	white    = rgb{1, 1, 1}
	shadow   = rgb{0x0B / 255.0, 0x25 / 255.0, 0x4D / 255.0}
)

// shade restituisce il colore del punto (x, y), in coordinate da 0 a 1 con
// l'origine in alto a sinistra; ok è false se il punto è trasparente.
func shade(x, y float64, withHole bool) (c rgb, ok bool) {
	if !inBackground(x, y) {
		return rgb{}, false
	}
	switch {
	case inWrench(x, y, withHole):
		return white, true
	case inWrench(x-0.025, y-0.035, withHole):
		// Ombra della chiave, leggermente spostata in basso a destra.
		return mix(gradient(y), shadow, 0.45), true
	}
	return gradient(y), true
}

// inBackground indica se il punto è dentro il quadrato arrotondato.
func inBackground(x, y float64) bool {
	const (
		margin = 1.0 / 32
		radius = 0.19
	)
	half := 0.5 - margin
	// Distanza con segno da un rettangolo con gli angoli arrotondati.
	qx := math.Abs(x-0.5) - (half - radius)
	qy := math.Abs(y-0.5) - (half - radius)
	outside := math.Hypot(math.Max(qx, 0), math.Max(qy, 0))
	inside := math.Min(math.Max(qx, qy), 0)
	return outside+inside-radius <= 0
}

// inWrench indica se il punto è dentro la chiave inglese, disposta in
// diagonale con la testa in alto a destra.
func inWrench(x, y float64, withHole bool) bool {
	// Coordinate ruotate di 45°: v corre lungo il manico verso la testa,
	// u di traverso.
	px, py := x-0.5, y-0.5
	v := (px - py) / math.Sqrt2
	u := (px + py) / math.Sqrt2

	const (
		handleStart = -0.31
		handleEnd   = 0.12
		handleHalf  = 0.075
		headCenter  = 0.17
		headRadius  = 0.19
		slotHalf    = 0.072
		slotBottom  = 0.20
		holeCenter  = -0.27
		holeRadius  = 0.028
	)

	inHandle := math.Hypot(u, v-clamp(v, handleStart, handleEnd)) <= handleHalf
	inHead := math.Hypot(u, v-headCenter) <= headRadius
	if !inHandle && !inHead {
		return false
	}
	// Apertura della testa, con il fondo arrotondato.
	if (math.Abs(u) < slotHalf && v > slotBottom) || math.Hypot(u, v-slotBottom) < slotHalf {
		return false
	}
	// Foro all'estremità del manico, visibile solo nelle dimensioni grandi.
	if withHole && math.Hypot(u, v-holeCenter) < holeRadius {
		return false
	}
	return true
}

// gradient restituisce il colore di sfondo all'altezza y.
func gradient(y float64) rgb {
	return mix(bgTop, bgBottom, y)
}

func mix(a, b rgb, t float64) rgb {
	return rgb{a.r + (b.r-a.r)*t, a.g + (b.g-a.g)*t, a.b + (b.b-a.b)*t}
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func to8(v float64) uint8 {
	return uint8(math.Round(clamp(v, 0, 1) * 255))
}

// encodeICO compone il file .ico: le immagini da pngMinSize in su sono
// salvate come PNG, le altre come bitmap a 32 bit con maschera, il formato
// letto da tutte le versioni di Windows.
func encodeICO(images []*image.NRGBA) ([]byte, error) {
	entries := make([][]byte, len(images))
	for i, img := range images {
		var err error
		if img.Rect.Dx() >= pngMinSize {
			var buf bytes.Buffer
			err = png.Encode(&buf, img)
			entries[i] = buf.Bytes()
		} else {
			entries[i] = encodeBMP(img)
		}
		if err != nil {
			return nil, err
		}
	}

	const (
		headerSize = 6
		entrySize  = 16
	)
	var out bytes.Buffer
	le := binary.LittleEndian
	w := func(v interface{}) { _ = binary.Write(&out, le, v) }

	// ICONDIR: riservato, tipo 1 (icona), numero di immagini.
	w(uint16(0))
	w(uint16(1))
	w(uint16(len(images)))

	offset := headerSize + entrySize*len(images)
	for i, img := range images {
		s := img.Rect.Dx()
		dim := uint8(s)
		if s >= 256 {
			dim = 0 // 0 significa 256
		}
		w(dim)        // larghezza
		w(dim)        // altezza
		w(uint8(0))   // colori della tavolozza
		w(uint8(0))   // riservato
		w(uint16(1))  // piani
		w(uint16(32)) // bit per pixel
		w(uint32(len(entries[i])))
		w(uint32(offset))
		offset += len(entries[i])
	}
	for _, e := range entries {
		out.Write(e)
	}
	return out.Bytes(), nil
}

// encodeBMP codifica img come bitmap DIB a 32 bit per un file .ico:
// intestazione con altezza doppia, pixel BGRA dal basso verso l'alto e
// maschera AND a 1 bit.
func encodeBMP(img *image.NRGBA) []byte {
	width, height := img.Rect.Dx(), img.Rect.Dy()
	maskStride := (width + 31) / 32 * 4
	pixelsSize := width * height * 4
	maskSize := maskStride * height

	var out bytes.Buffer
	le := binary.LittleEndian
	w := func(v interface{}) { _ = binary.Write(&out, le, v) }

	// BITMAPINFOHEADER.
	w(uint32(40))
	w(int32(width))
	w(int32(height * 2)) // immagine più maschera
	w(uint16(1))         // piani
	w(uint16(32))        // bit per pixel
	w(uint32(0))         // nessuna compressione
	w(uint32(pixelsSize + maskSize))
	w(int32(0))
	w(int32(0))
	w(uint32(0))
	w(uint32(0))

	for y := height - 1; y >= 0; y-- {
		for x := 0; x < width; x++ {
			c := img.NRGBAAt(x, y)
			out.Write([]byte{c.B, c.G, c.R, c.A})
		}
	}
	// Nella maschera un bit a 1 indica un pixel trasparente.
	mask := make([]byte, maskStride)
	for y := height - 1; y >= 0; y-- {
		for i := range mask {
			mask[i] = 0
		}
		for x := 0; x < width; x++ {
			if img.NRGBAAt(x, y).A == 0 {
				mask[x/8] |= 0x80 >> (x % 8)
			}
		}
		out.Write(mask)
	}
	return out.Bytes()
}
