package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

const (
	canvasX   = 32
	canvasY   = 32
	canvasW   = 10
	canvasH   = 20
	blockSize = 25
)

type block struct {
	color string
	x, y  int32
}

var quit bool
var pause bool
var gameover bool

var piece []*block
var pieceType string
var nextPiece string
var pieces = []string{"I", "O", "T", "L", "J", "S", "Z"}
var canvas [canvasW][canvasH]*block

var cmd string
var score int
var level = 1
var lastMove time.Time = time.Now()
var scores map[string]int
var highScores []string

var renderer *sdl.Renderer
var font *ttf.Font

func color(c string) (r, g, b, a uint8) {
	switch c {
	case "red":
		r = 255
	case "green":
		g = 255
	case "blue":
		b = 255
	case "darkgreen":
		g = 128
	case "cyan":
		g, b = 255, 255
	case "magenta":
		r, b = 255, 255
	case "yellow":
		r, g = 255, 255
	case "gray":
		r, g, b = 128, 128, 128
	case "white":
		r, g, b = 255, 255, 255
	case "pink":
		r, g, b = 255, 200, 215
	}
	return
}

func rect(x, y int32) *sdl.Rect {
	x, y = x*blockSize, y*blockSize
	return &sdl.Rect{x + canvasX, y + canvasY, blockSize, blockSize}
}

func (b *block) render() {
	renderer.SetDrawColor(color(b.color))
	renderer.FillRect(rect(b.x, b.y))
	renderer.SetDrawColor(color("black"))
	renderer.DrawRect(rect(b.x, b.y))
}

func newPiece(typ string) []*block {
	switch typ {
	case "I":
		return []*block{
			{"red", 4, 1},
			{"red", 4, 0},
			{"red", 4, 2},
			{"red", 4, 3},
		}
	case "O":
		return []*block{
			{"cyan", 4, 0},
			{"cyan", 3, 0},
			{"cyan", 3, 1},
			{"cyan", 4, 1},
		}
	case "T":
		return []*block{
			{"gray", 4, 1},
			{"gray", 4, 0},
			{"gray", 3, 1},
			{"gray", 5, 1},
		}
	case "L":
		return []*block{
			{"yellow", 4, 1},
			{"yellow", 4, 0},
			{"yellow", 4, 2},
			{"yellow", 5, 2},
		}
	case "J":
		return []*block{
			{"magenta", 4, 1},
			{"magenta", 4, 0},
			{"magenta", 4, 2},
			{"magenta", 3, 2},
		}
	case "S":
		return []*block{
			{"blue", 4, 0},
			{"blue", 5, 0},
			{"blue", 4, 1},
			{"blue", 3, 1},
		}
	case "Z":
		return []*block{
			{"darkgreen", 4, 0},
			{"darkgreen", 3, 0},
			{"darkgreen", 4, 1},
			{"darkgreen", 5, 1},
		}
	}
	return nil
}

func createPiece() {
	pieceType = nextPiece
	piece = newPiece(pieceType)
	nextPiece = pieces[sdl.GetTicks()%7]
	for _, b := range piece {
		if canvas[b.x][b.y] != nil {
			pause = true
			gameover = true
			updateCloud()
			return
		}
	}
}

func updateCloud() {
	user := os.Getenv("USERNAME")
	urlfmt := "http://dogankurt.com/tetris?user=%s&score=%d"
	url := fmt.Sprintf(urlfmt, user, score)
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	resp.Body.Close()
	time.Sleep(500 * time.Millisecond)
	getScores()
}

func getScores() {
	resp, err := http.Get("http://dogankurt.com/tetrisscore")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	scr := make(map[string]int)
	err = json.Unmarshal(data, &scr)
	if err != nil {
		return
	}
	scores = scr
	names := []string{}
	for name := range scores {
		names = append(names, name)
	}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if scores[names[i]] < scores[names[j]] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	highScores = names
}

func reset() {
	pause = false
	gameover = false
	cmd = ""
	score = 0
	level = 1
	lastMove = time.Now()
	for x := range canvas {
		for y := range canvas[x] {
			canvas[x][y] = nil
		}
	}
	createPiece()
	sdl.FlushEvent(sdl.KEYDOWN)
}

func input() {
	for ev := sdl.PollEvent(); ev != nil; ev = sdl.PollEvent() {
		switch t := ev.(type) {
		case *sdl.QuitEvent:
			quit = true
		case *sdl.KeyboardEvent:
			if t.Type != sdl.KEYDOWN {
				break
			}
			switch key := t.Keysym.Sym; key {
			case sdl.K_ESCAPE:
				quit = true
			case sdl.K_UP:
				cmd = "up"
			case sdl.K_DOWN:
				cmd = "down"
			case sdl.K_LEFT:
				cmd = "left"
			case sdl.K_RIGHT:
				cmd = "right"
			case sdl.K_p:
				pause = !pause
			case sdl.K_RETURN:
				if gameover {
					reset()
				}
			}
		}
	}
}

func pieceCopy() []*block {
	b := []block{
		*piece[0],
		*piece[1],
		*piece[2],
		*piece[3],
	}
	return []*block{&b[0], &b[1], &b[2], &b[3]}
}

func legalState(p []*block) bool {
	for _, b := range p {
		if b.x < 0 || b.x > canvasW-1 {
			return false
		}
		if b.y < 0 || b.y > canvasH-1 {
			return false
		}
		if canvas[b.x][b.y] != nil {
			return false
		}
	}
	return true
}

func movePiece(way string) bool {
	p := pieceCopy()
	switch way {
	case "right":
		for _, b := range p {
			b.x++
		}
	case "left":
		for _, b := range p {
			b.x--
		}
	case "down":
		for _, b := range p {
			b.y++
		}
	}
	if legalState(p) {
		piece = p
		return true
	}
	return false
}

func rotatePiece() {
	if pieceType == "O" {
		return
	}
	p := pieceCopy()
	x, y := p[0].x, p[0].y
	for _, b := range p {
		rx, ry := b.x-x, b.y-y
		rx, ry = ry, rx*-1
		b.x, b.y = rx+x, ry+y
	}
	if legalState(p) {
		piece = p
		return
	}
	tryAgain := false
	for _, b := range p {
		switch {
		case b.x < 0:
			movePiece("right")
			tryAgain = true
		case b.x > canvasW-1:
			movePiece("left")
			tryAgain = true
		case b.y < 0:
			movePiece("down")
			tryAgain = true
		}
	}
	if tryAgain {
		rotatePiece()
	}
}

func tetris(h int) {
	for x := 0; x < canvasW; x++ {
		canvas[x][h] = nil
	}
	for y := h; y >= 0; y-- {
		for x := 0; x < canvasW; x++ {
			b := canvas[x][y]
			if b != nil {
				b.y++
				canvas[x][y] = nil
				canvas[b.x][b.y] = b
			}
		}
	}
}

func mergePiece() {
	for _, b := range piece {
		canvas[b.x][b.y] = b
	}
	n := 0
	for y := 0; y < canvasH; y++ {
		boom := true
		for x := 0; x < canvasW; x++ {
			if canvas[x][y] == nil {
				boom = false
				break
			}
		}
		if boom {
			tetris(y)
			n++
		}
	}
	switch n {
	case 1:
		score += 45 * level
	case 2:
		score += 100 * level
	case 3:
		score += 300 * level
	case 4:
		score += 1000 * level
	}
}

func update() {
	delta := int64(1000/level) * int64(time.Millisecond)
	if time.Since(lastMove).Nanoseconds() >= delta {
		if !movePiece("down") {
			mergePiece()
			createPiece()
		}
		lastMove = time.Now()
	}
	switch cmd {
	case "up":
		rotatePiece()
	case "down":
		for i := 0; i < canvasH; i++ {
			if !movePiece(cmd) {
				break
			}
		}
		mergePiece()
		createPiece()
	case "left", "right":
		movePiece(cmd)
	}
	cmd = ""
	switch {
	case score > 5000:
		level = 2
	case score > 50000:
		level = 3
	case score > 500000:
		level = 4
	case score > 1000000:
		level = 5
	}
}

func renderThickRect(r *sdl.Rect, n int32, clr string) {
	renderer.SetDrawColor(color(clr))
	for i := int32(0); i < n; i++ {
		rect := &sdl.Rect{r.X + i, r.Y + i, r.W - 2*i, r.H - 2*i}
		renderer.DrawRect(rect)
	}
}

func renderCanvas() {
	var x, y, w, h int32 = canvasX, canvasY, canvasW, canvasH
	w *= blockSize
	h *= blockSize
	renderer.SetDrawColor(color("pink"))
	for x := range canvas {
		for y := range canvas[x] {
			renderer.DrawRect(rect(int32(x), int32(y)))
		}
	}
	renderThickRect(&sdl.Rect{x - 6, y - 6, w + 12, h + 12}, 4, "green")
	renderThickRect(&sdl.Rect{x - 2, y - 2, w + 4, h + 4}, 4, "black")
	for x := range canvas {
		for y := range canvas[x] {
			if canvas[x][y] != nil {
				canvas[x][y].render()
			}
		}
	}
}

func renderWindows() {
	// render next piece
	r := &sdl.Rect{340, 60, 125, 150}
	renderer.SetDrawColor(color("pink"))
	renderer.FillRect(r)
	p := newPiece(nextPiece)
	x, y := p[0].x, p[0].y
	for _, b := range p {
		r := rect(b.x-x, b.y-y)
		r.X += 360
		r.Y += 80
		renderer.SetDrawColor(color(b.color))
		renderer.FillRect(r)
		renderer.SetDrawColor(color("black"))
		renderer.DrawRect(r)
	}
	renderThickRect(r, 4, "black")

	// render score
	r = &sdl.Rect{500, 60, 150, 55}
	renderer.SetDrawColor(color("pink"))
	renderer.FillRect(r)
	renderText(fmt.Sprintf("%d", score), "black", 510, 65)
	renderThickRect(r, 4, "black")

	// render high scores
	r = &sdl.Rect{340, 250, 310, 280}
	renderer.SetDrawColor(color("pink"))
	renderer.FillRect(r)
	for i, name := range highScores {
		s := fmt.Sprintf("%d", scores[name])
		renderText(name, "black", 350, int32(260+i*40))
		renderText(s, "red", 520, int32(260+i*40))
	}
	renderThickRect(r, 4, "black")

	switch {
	case gameover:
		r := &sdl.Rect{275, 230, 225, 80}
		renderer.SetDrawColor(color("pink"))
		renderer.FillRect(r)
		renderText("GAME OVER", "red", 290, 250)
		renderThickRect(r, 4, "red")
	case pause:
		r := &sdl.Rect{275, 230, 150, 80}
		renderer.SetDrawColor(color("darkgreen"))
		renderer.FillRect(r)
		renderText("Paused", "green", 295, 250)
		renderThickRect(r, 4, "black")
	}
}

func renderText(text string, clr string, x, y int32) {
	r, g, b, a := color(clr)
	c := sdl.Color{r, g, b, a}
	stext, err := font.RenderUTF8Blended(text, c)
	if err != nil {
		panic(err)
	}
	defer stext.Free()
	texture, err := renderer.CreateTextureFromSurface(stext)
	if err != nil {
		panic(err)
	}
	defer texture.Destroy()
	renderer.Copy(texture, nil, &sdl.Rect{x, y, stext.W, stext.H})
}

func render() {
	renderer.SetDrawColor(color("white"))
	renderer.Clear()
	renderCanvas()
	for _, b := range piece {
		b.render()
	}
	renderWindows()
	renderer.Present()
}

func createWindow(name string) *sdl.Window {
	var undef int32 = sdl.WINDOWPOS_UNDEFINED
	var wndtype uint32 = sdl.WINDOW_SHOWN
	w, h := int32(680), int32(560)
	window, err := sdl.CreateWindow(name, undef, undef, w, h, wndtype)
	if err != nil {
		panic(err)
	}
	return window
}

func createRenderer(win *sdl.Window) *sdl.Renderer {
	var flag uint32 = sdl.RENDERER_ACCELERATED | sdl.RENDERER_PRESENTVSYNC
	renderer, err := sdl.CreateRenderer(win, -1, flag)
	if err != nil {
		panic(err)
	}
	return renderer
}

func init() {
	runtime.LockOSThread()
}

func main() {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	if err := ttf.Init(); err != nil {
		panic(err)
	}
	defer ttf.Quit()

	fnt, err := ttf.OpenFont("georgia.ttf", 32)
	if err != nil {
		panic(err)
	}
	font = fnt

	window := createWindow("tetris")
	defer window.Destroy()

	renderer = createRenderer(window)
	defer renderer.Destroy()

	getScores()

	nextPiece = pieces[sdl.GetTicks()%7]
	createPiece()

	for !quit {
		input()
		if !pause {
			update()
		}
		render()
	}
}
