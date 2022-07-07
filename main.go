package main

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"math/rand"
	"strings"
	"time"

	"encoding/json"

	"github.com/atotto/clipboard"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

const (
	SIZE_X = 400
	SIZE_Y = 200
	ZOOM   = 3

	MUTATE     = 50
	LEN_GENOME = 50

	MAX_GENOME = 5000
	MAX_MOLD   = 20000

	RIGHT = 0
	TOP   = 1
	LEFT  = 2

	SPORE = -1
	NONE  = -2

	NUM = LEN_GENOME * 3
	R   = LEN_GENOME*3 + 1
	G   = LEN_GENOME*3 + 2
	B   = LEN_GENOME*3 + 3

	ENERGY_LIGHT_MAX = 20
	ENERGY_DAY       = 10
	ENERGY_MOLD      = 5000

	TIME_CELL = 200
)

// time and pause
var TIME = 0
var PAUSE = false

// light parameter
var ENERGY_LIGHT = 18
var ENERGY_VISIAL_TIME = 0

// for load gen in clipboard
var mouse_gen [LEN_GENOME*3 + 4]int
var MOUSE_COPY_TIME = -1000
var MOUSE_LOAD_TIME = -1000
var START_HELLO_TIME = true

// font
var NormalFont font.Face

// position molds
var cells [SIZE_X][SIZE_Y]Cell
var molds [MAX_MOLD]Mold

// gen n is in the positions 3n, 3n+1, 3n+2
// 3n - left growth
// 3n+1 - top growth
// 3n+2 - rigrt growth
// no bottom growth
var genomes [MAX_GENOME][LEN_GENOME*3 + 4]int

type Cell struct {
	mold   int
	n      int
	dx, dy int
	time   int
}

type Mold struct {
	genome int
	energy int
	num    int
	color  int
}

type Game struct {
	pixels []byte
}

// generate one gen
func rand_gen() int {
	return (rand.Intn(LEN_GENOME+2))*rand.Intn(2) - 2
}

// generate random color for genome
func rand_color(genome int) {
	genomes[genome][R] = 10 + rand.Intn(130)
	genomes[genome][G] = 10 + rand.Intn(130)
	genomes[genome][B] = 10 + rand.Intn(130)
}

// x mod SIZE_X
func mod_x(x int) int {
	if x >= SIZE_X {
		return 0
	}
	if x < 0 {
		return SIZE_X - 1
	}
	return x
}

// y mod SIZE_Y
func mod_y(y int) int {
	if y >= SIZE_Y {
		return 0
	}
	if y < 0 {
		return SIZE_Y - 1
	}
	return y
}

// found i new genome
func found_new_genome() int {
	new_genome := 1
	for new_genome < MAX_GENOME {
		if genomes[new_genome][NUM] == 0 {
			break
		}
		new_genome++
	}
	// check num genome
	if new_genome >= MAX_GENOME {
		fmt.Println("Panic! Too much genome!")
	}
	return new_genome
}

// found i new mold
func found_new_mold() int {
	new_mold := 1
	for new_mold < MAX_MOLD {
		if molds[new_mold].num == 0 {
			break
		}
		new_mold++
	}
	// chech num mold
	if new_mold >= MAX_MOLD {
		fmt.Println("Panic! Too much mold!")
	}
	return new_mold
}

// func delete_all() {
// 	fmt.Println("Delete all!")
// 	// delete all genomes
// 	for i := 0; i < MAX_GENOME; i++ {
// 		genomes[i][NUM] = 0
// 	}
// 	// delete all molds
// 	for i := 0; i < MAX_MOLD; i++ {
// 		molds[i].num = 0
// 	}
// 	// delete all cells
// 	for i := 0; i < SIZE_X; i++ {
// 		for j := 0; j < SIZE_Y; j++ {
// 			cells[i][j].mold = 0
// 		}
// 	}
// }

func generate_new_mold(x, y int) {
	if cells[x][y].mold == 0 {
		// found i for new genome and new mold
		new_genome := found_new_genome()
		new_mold := found_new_mold()

		if new_genome < MAX_GENOME && new_mold < MAX_MOLD {
			// generate new genome
			for i := 0; i < LEN_GENOME*3; i++ {
				genomes[new_genome][i] = rand_gen()
			}
			genomes[new_genome][NUM] = 1
			rand_color(new_genome)

			// create new mold and cell
			molds[new_mold] = Mold{new_genome, ENERGY_LIGHT * 10, 1, rand.Intn(60)}
			cells[x][y] = Cell{new_mold, 0, 0, 1, 0}
		}
	}
}

func load_genom(x, y int) {
	// found i for new genome and new mold
	new_genome := found_new_genome()
	new_mold := found_new_mold()

	if new_genome < MAX_GENOME && new_mold < MAX_MOLD {
		// copy mouse gemone
		for i := 0; i < LEN_GENOME*3+4; i++ {
			genomes[new_genome][i] = mouse_gen[i]
		}
		genomes[new_genome][NUM] = 1

		// creadte mold and cell
		molds[new_mold] = Mold{new_genome, ENERGY_LIGHT * 10, 1, rand.Intn(60)}
		cells[x][y] = Cell{new_mold, 0, 0, 1, 0}
	}
}

func create_new_mold(x, y int) {
	// found i last genome and last mold
	last_mold := cells[x][y].mold
	last_genome := molds[last_mold].genome

	// found i new genome and new mold
	new_mold := found_new_mold()
	new_genome := found_new_genome() // it may not be needed

	if new_genome < MAX_GENOME && new_mold < MAX_MOLD {
		if rand.Intn(MUTATE) == 0 {
			// copy genome and mutation
			new_genome := found_new_genome()
			for i := 0; i < LEN_GENOME*3; i++ {
				genomes[new_genome][i] = genomes[last_genome][i]
			}
			genomes[new_genome][NUM] = 1
			rand_color(new_genome)

			// mutate genome (5 gens)
			if rand.Intn(MUTATE) == 0 {
				for i := 0; i < 10; i++ {
					genomes[new_genome][rand.Intn(LEN_GENOME*3)] = rand_gen()
				}
			}
			// create new mold
			molds[new_mold] = Mold{new_genome, cells[x][y].time, 1, rand.Intn(60)}
		} else {
			// create new mold
			genomes[last_genome][NUM]++
			molds[new_mold] = Mold{last_genome, cells[x][y].time, 1, rand.Intn(60)}
		}

		// creade cell
		cells[x][y].mold = new_mold
		cells[x][y].n = 0
		cells[x][y].time = 0
	} else {
		cells[x][y].mold = 0
	}
}

func add_cell(x, y, x2, y2, n int) {
	if cells[x2][y2].mold == 0 {
		// if it is spore, it takes energy
		if n != SPORE || molds[cells[x][y].mold].energy > ENERGY_MOLD {
			// mold use energy
			if n == SPORE {
				molds[cells[x][y].mold].energy -= ENERGY_MOLD
			}
			// mold add num
			molds[cells[x][y].mold].num++

			// add cell
			cells[x2][y2].mold = cells[x][y].mold
			cells[x2][y2].n = n
			cells[x2][y2].time = 0

			// direction cell x
			if x == x2 {
				cells[x2][y2].dx = 0
				if y < y2 || y2 == 0 {
					cells[x2][y2].dy = 1
				} else {
					cells[x2][y2].dy = -1
				}
			}
			// direction cell y
			if y == y2 {
				cells[x2][y2].dy = 0
				if x < x2 || x2 == 0 {
					cells[x2][y2].dx = 1
				} else {
					cells[x2][y2].dx = -1
				}
			}
		}
	}
}

func neitherhood(x, y, dx, dy, dir int) (int, int) {
	// top direction
	if (dx == 1 && dir == 0) || (dx == -1 && dir == 2) || (dy == 1 && dir == 1) {
		return x, mod_y(y + 1)
	}
	// right direction
	if (dx == 1 && dir == 1) || (dy == 1 && dir == 2) || (dy == -1 && dir == 0) {
		return mod_x(x + 1), y
	}
	// bottom direction
	if (dx == 1 && dir == 2) || (dx == -1 && dir == 0) || (dy == -1 && dir == 1) {
		return x, mod_y(y - 1)
	}
	// left direction
	if (dx == -1 && dir == 1) || (dy == 1 && dir == 0) || (dy == -1 && dir == 2) {
		return mod_x(x - 1), y
	}
	// panic!
	fmt.Println("Problem neitherhood (panic!)")
	return 0, 0
}

func growth_cell(x, y int) {
	if cells[x][y].n != SPORE {
		// growth cell on four direction
		for dir := 0; dir < 3; dir++ {
			// new cell gen in the genome
			next_n := genomes[molds[cells[x][y].mold].genome][cells[x][y].n*3+dir]
			if next_n != NONE {
				x2, y2 := neitherhood(x, y, cells[x][y].dx, cells[x][y].dy, dir)
				add_cell(x, y, x2, y2, next_n)
			}
		}
	}
}

func ph(x2, y2 int) (energy int) {
	// empty cell next to it gives energy
	if cells[x2][y2].mold == 0 {
		return ENERGY_LIGHT
	}
	return 0
}

func photosynthesis(x, y int) {
	// neitherhood on four direction
	molds[cells[x][y].mold].energy += ph(mod_x(x+1), y) + ph(mod_x(x-1), y) + ph(x, mod_y(y+1)) + ph(x, mod_y(y-1))
}

func delete_cell(x, y int) {
	delete_mold := cells[x][y].mold
	delete_genome := molds[delete_mold].genome

	// delete cell
	if cells[x][y].n == SPORE {
		create_new_mold(x, y)
	} else {
		cells[x][y].mold = 0
	}

	// delete nil mold and nil genome
	molds[delete_mold].num--
	if molds[delete_mold].num <= 0 {
		genomes[delete_genome][NUM]--
	}
}

func update() {
	if !PAUSE {
		// photosynthesis, food and aging
		for x := 0; x < SIZE_X; x++ {
			for y := 0; y < SIZE_Y; y++ {
				if cells[x][y].mold != 0 {
					cells[x][y].time++
					molds[cells[x][y].mold].energy -= ENERGY_DAY * (1 + int(cells[x][y].time/TIME_CELL))
					photosynthesis(x, y)
				}
			}
		}

		// growth cells
		for x := 0; x < SIZE_X; x++ {
			for y := 0; y < SIZE_Y; y++ {
				if cells[x][y].mold != 0 {
					if cells[x][y].time > 0 {
						growth_cell(x, y)
					}
				}
			}
		}

		// delete cells
		for x := 0; x < SIZE_X; x++ {
			for y := 0; y < SIZE_Y; y++ {
				if cells[x][y].mold != 0 {
					if molds[cells[x][y].mold].energy <= 0 {
						delete_cell(x, y)
					}
				}
			}
		}
	}
}

func key_press() {
	// gereration new molds
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		for i := 0; i < 300; i++ {
			generate_new_mold(rand.Intn(SIZE_X), rand.Intn(SIZE_Y))
		}
		START_HELLO_TIME = false
	}

	// on/off pause
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		PAUSE = !PAUSE
	}

	// energy light
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		ENERGY_LIGHT--
		if ENERGY_LIGHT < 0 {
			ENERGY_LIGHT = 0
		}
		ENERGY_VISIAL_TIME = TIME
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		ENERGY_LIGHT++
		if ENERGY_LIGHT > ENERGY_LIGHT_MAX {
			ENERGY_LIGHT = ENERGY_LIGHT_MAX
		}
		ENERGY_VISIAL_TIME = TIME
	}

	// print info
	// if inpututil.IsKeyJustPressed(ebiten.KeyI) {
	// 	num_g := 0
	// 	for i := 0; i < MAX_GENOME; i++ {
	// 		if genomes[i][NUM] != 0 {
	// 			num_g++
	// 		}
	// 	}
	// 	num_m := 0
	// 	for i := 0; i < MAX_MOLD; i++ {
	// 		if molds[i].num != 0 {
	// 			num_m++
	// 		}
	// 	}
	// 	fmt.Println("time", TIME, "genome", num_g, "mold", num_m)
	// }

	// delete all
	// if inpututil.IsKeyJustPressed(ebiten.KeyD) {
	// 	delete_all()
	// }

	// // FPS
	// if inpututil.IsKeyJustPressed(ebiten.KeyF) {
	// 	fmt.Println(ebiten.CurrentFPS())
	// }
}

func mouse_click() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// cursor position
		x, y := ebiten.CursorPosition()
		x = int(x / ZOOM)
		y = int(y / ZOOM)

		// check position
		if 0 <= x && x < SIZE_X && 0 <= y && y < SIZE_Y {
			// copy genom or add new mold next cursor
			if cells[x][y].mold != 0 {
				save, _ := json.Marshal(genomes[molds[cells[x][y].mold].genome])
				clipboard.WriteAll(string(save)) //  text genome save in clipboard
				fmt.Println("genome saved")
				MOUSE_COPY_TIME = TIME
				MOUSE_LOAD_TIME = -1000
			} else {
				save, _ := clipboard.ReadAll()
				err := json.Unmarshal([]byte(save), &mouse_gen) // text genome load in mouse_gen
				if err != nil {
					// no panic
					fmt.Println("problem clickboard (no panic)")
				} else {
					load_genom(x, y)
					fmt.Println("genome loaded")
					MOUSE_LOAD_TIME = TIME
					MOUSE_COPY_TIME = -1000
					START_HELLO_TIME = false
				}
			}
		} else {
			fmt.Println("Problem mouse click (no panic)")
		}
	}
}

func (g *Game) Update() error {
	TIME++
	mouse_click()
	key_press()
	update()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// create pixels
	if g.pixels == nil {
		g.pixels = make([]byte, SIZE_X*SIZE_Y*ZOOM*ZOOM*4)
	}

	// draw cells
	for x := 0; x < SIZE_X; x++ {
		for y := 0; y < SIZE_Y; y++ {
			if cells[x][y].mold != 0 {
				// draw cell  spore
				color_r := byte(genomes[molds[cells[x][y].mold].genome][R] + molds[cells[x][y].mold].color)
				color_g := byte(genomes[molds[cells[x][y].mold].genome][G] + molds[cells[x][y].mold].color)
				color_b := byte(genomes[molds[cells[x][y].mold].genome][B] + molds[cells[x][y].mold].color)
				for i := 0; i < ZOOM; i++ {
					for j := 0; j < ZOOM; j++ {
						pic := ((y*ZOOM+j)*SIZE_X*ZOOM + x*ZOOM + i) * 4
						g.pixels[pic] = color_r
						g.pixels[pic+1] = color_g
						g.pixels[pic+2] = color_b
						g.pixels[pic+3] = 0xff
					}
				}
				// draw spore
				if cells[x][y].n == SPORE {
					for i := 1; i < ZOOM-1; i++ {
						for j := 1; j < ZOOM-1; j++ {
							pic := ((y*ZOOM+j)*SIZE_X*ZOOM + x*ZOOM + i) * 4
							g.pixels[pic] = 0
							g.pixels[pic+1] = 0
							g.pixels[pic+2] = 0
							g.pixels[pic+3] = 0xff
						}
					}
				}
			} else {
				// draw nil cell
				for i := 0; i < ZOOM; i++ {
					for j := 0; j < ZOOM; j++ {
						pic := ((y*ZOOM+j)*SIZE_X*ZOOM + x*ZOOM + i) * 4
						g.pixels[pic] = 0
						g.pixels[pic+1] = 0
						g.pixels[pic+2] = 0
						g.pixels[pic+3] = 0
					}
				}
			}
		}
	}

	// graw light
	if TIME-ENERGY_VISIAL_TIME < 120 {
		for x := 20; x < 20+ENERGY_LIGHT*16; x++ {
			for y := 20; y < 40; y++ {
				pic := (y*SIZE_X*ZOOM + x) * 4
				g.pixels[pic] = 0xc0
				g.pixels[pic+1] = 0xc3
				g.pixels[pic+2] = 0x50
				g.pixels[pic+3] = 0xff
			}
		}
		for x := 20 + ENERGY_LIGHT*16; x < 20+ENERGY_LIGHT_MAX*16; x++ {
			for y := 20; y < 40; y++ {
				pic := (y*SIZE_X*ZOOM + x) * 4
				g.pixels[pic] = 0x60
				g.pixels[pic+1] = 0x60
				g.pixels[pic+2] = 0x60
				g.pixels[pic+3] = 0xff
			}
		}
	}

	// screen output
	screen.ReplacePixels(g.pixels)

	// graw text light
	if TIME-ENERGY_VISIAL_TIME < 120 {
		text.Draw(screen, fmt.Sprint(ENERGY_LIGHT), NormalFont, 20+ENERGY_LIGHT_MAX*8, 37, color.Black)
	}

	// graw copy/load
	if TIME-MOUSE_COPY_TIME < 120 {
		text.Draw(screen, "Genome saved in clipboard.", NormalFont, 20, 60, color.White)
	}
	if TIME-MOUSE_LOAD_TIME < 120 {
		text.Draw(screen, "Genome loaded from clipboard.", NormalFont, 20, 60, color.White)
	}

	// graw start hello
	if START_HELLO_TIME {
		text.Draw(screen, "Press G to generate new molds", NormalFont, int(SIZE_X*ZOOM/2)-150, int(SIZE_Y*ZOOM/2), color.White)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	// size window
	return SIZE_X * ZOOM, SIZE_Y * ZOOM
}

func main() {
	// seed for random
	rand.Seed(time.Now().UnixNano())

	// create window
	ebiten.SetWindowSize(SIZE_X*ZOOM, SIZE_Y*ZOOM)
	ebiten.SetWindowTitle("Cute mold")
	//ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)

	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(logoData))
	logoImage, _, err := image.Decode(reader)
	if err != nil {
		panic(err)
	}
	ebiten.SetWindowIcon([]image.Image{logoImage})

	// font for text on the screen
	tt, _ := opentype.Parse(fonts.MPlus1pRegular_ttf)
	NormalFont, _ = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    20,
		DPI:     72,
		Hinting: font.HintingFull,
	})

	// game run
	game := &Game{}
	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}

const logoData = `
iVBORw0KGgoAAAANSUhEUgAAAUAAAAFACAYAAADNkKWqAAABN2lDQ1BBZG9iZSBSR0IgKDE5OTgp
AAAokZWPv0rDUBSHvxtFxaFWCOLgcCdRUGzVwYxJW4ogWKtDkq1JQ5ViEm6uf/oQjm4dXNx9AidH
wUHxCXwDxamDQ4QMBYvf9J3fORzOAaNi152GUYbzWKt205Gu58vZF2aYAoBOmKV2q3UAECdxxBjf
7wiA10277jTG+38yH6ZKAyNguxtlIYgK0L/SqQYxBMygn2oQD4CpTto1EE9AqZf7G1AKcv8ASsr1
fBBfgNlzPR+MOcAMcl8BTB1da4Bakg7UWe9Uy6plWdLuJkEkjweZjs4zuR+HiUoT1dFRF8jvA2Ax
H2w3HblWtay99X/+PRHX82Vun0cIQCw9F1lBeKEuf1UYO5PrYsdwGQ7vYXpUZLs3cLcBC7dFtlqF
8hY8Dn8AwMZP/fNTP8gAAAAJcEhZcwAADsQAAA7EAZUrDhsAAAezaVRYdFhNTDpjb20uYWRvYmUu
eG1wAAAAAAA8P3hwYWNrZXQgYmVnaW49Iu+7vyIgaWQ9Ilc1TTBNcENlaGlIenJlU3pOVGN6a2M5
ZCI/PiA8eDp4bXBtZXRhIHhtbG5zOng9ImFkb2JlOm5zOm1ldGEvIiB4OnhtcHRrPSJBZG9iZSBY
TVAgQ29yZSA1LjYtYzE0NSA3OS4xNjM0OTksIDIwMTgvMDgvMTMtMTY6NDA6MjIgICAgICAgICI+
IDxyZGY6UkRGIHhtbG5zOnJkZj0iaHR0cDovL3d3dy53My5vcmcvMTk5OS8wMi8yMi1yZGYtc3lu
dGF4LW5zIyI+IDxyZGY6RGVzY3JpcHRpb24gcmRmOmFib3V0PSIiIHhtbG5zOnhtcD0iaHR0cDov
L25zLmFkb2JlLmNvbS94YXAvMS4wLyIgeG1sbnM6eG1wTU09Imh0dHA6Ly9ucy5hZG9iZS5jb20v
eGFwLzEuMC9tbS8iIHhtbG5zOnN0RXZ0PSJodHRwOi8vbnMuYWRvYmUuY29tL3hhcC8xLjAvc1R5
cGUvUmVzb3VyY2VFdmVudCMiIHhtbG5zOmRjPSJodHRwOi8vcHVybC5vcmcvZGMvZWxlbWVudHMv
MS4xLyIgeG1sbnM6cGhvdG9zaG9wPSJodHRwOi8vbnMuYWRvYmUuY29tL3Bob3Rvc2hvcC8xLjAv
IiB4bXA6Q3JlYXRvclRvb2w9IkFkb2JlIFBob3Rvc2hvcCBDQyAyMDE5IChXaW5kb3dzKSIgeG1w
OkNyZWF0ZURhdGU9IjIwMjItMDctMDZUMTI6MTE6MTUrMDM6MDAiIHhtcDpNZXRhZGF0YURhdGU9
IjIwMjItMDctMDdUMTI6MTU6MTgrMDM6MDAiIHhtcDpNb2RpZnlEYXRlPSIyMDIyLTA3LTA3VDEy
OjE1OjE4KzAzOjAwIiB4bXBNTTpJbnN0YW5jZUlEPSJ4bXAuaWlkOjMxMzdmOTdmLTBkZDgtMjY0
NS1iNjIyLWY1MDE3MTZhOWJkNSIgeG1wTU06RG9jdW1lbnRJRD0iYWRvYmU6ZG9jaWQ6cGhvdG9z
aG9wOmVlMzdiYjZmLTljZDktOGU0Zi1iMWU2LWI2ZmU2YWQ5NmFlMSIgeG1wTU06T3JpZ2luYWxE
b2N1bWVudElEPSJ4bXAuZGlkOjEyZGM1ZmNhLWMwMDgtMjM0Ni05ZTAxLThjMmYzNTEwYmY0YSIg
ZGM6Zm9ybWF0PSJpbWFnZS9wbmciIHBob3Rvc2hvcDpDb2xvck1vZGU9IjMiIHBob3Rvc2hvcDpJ
Q0NQcm9maWxlPSJBZG9iZSBSR0IgKDE5OTgpIj4gPHhtcE1NOkhpc3Rvcnk+IDxyZGY6U2VxPiA8
cmRmOmxpIHN0RXZ0OmFjdGlvbj0iY3JlYXRlZCIgc3RFdnQ6aW5zdGFuY2VJRD0ieG1wLmlpZDox
MmRjNWZjYS1jMDA4LTIzNDYtOWUwMS04YzJmMzUxMGJmNGEiIHN0RXZ0OndoZW49IjIwMjItMDct
MDZUMTI6MTE6MTUrMDM6MDAiIHN0RXZ0OnNvZnR3YXJlQWdlbnQ9IkFkb2JlIFBob3Rvc2hvcCBD
QyAyMDE5IChXaW5kb3dzKSIvPiA8cmRmOmxpIHN0RXZ0OmFjdGlvbj0ic2F2ZWQiIHN0RXZ0Omlu
c3RhbmNlSUQ9InhtcC5paWQ6OGY4MWJhNzUtYjA4MS1lZDQ2LTg1NmEtMjIzNWRlN2RmOTdlIiBz
dEV2dDp3aGVuPSIyMDIyLTA3LTA2VDEyOjExOjE1KzAzOjAwIiBzdEV2dDpzb2Z0d2FyZUFnZW50
PSJBZG9iZSBQaG90b3Nob3AgQ0MgMjAxOSAoV2luZG93cykiIHN0RXZ0OmNoYW5nZWQ9Ii8iLz4g
PHJkZjpsaSBzdEV2dDphY3Rpb249InNhdmVkIiBzdEV2dDppbnN0YW5jZUlEPSJ4bXAuaWlkOjMx
MzdmOTdmLTBkZDgtMjY0NS1iNjIyLWY1MDE3MTZhOWJkNSIgc3RFdnQ6d2hlbj0iMjAyMi0wNy0w
N1QxMjoxNToxOCswMzowMCIgc3RFdnQ6c29mdHdhcmVBZ2VudD0iQWRvYmUgUGhvdG9zaG9wIEND
IDIwMTkgKFdpbmRvd3MpIiBzdEV2dDpjaGFuZ2VkPSIvIi8+IDwvcmRmOlNlcT4gPC94bXBNTTpI
aXN0b3J5PiA8cGhvdG9zaG9wOkRvY3VtZW50QW5jZXN0b3JzPiA8cmRmOkJhZz4gPHJkZjpsaT5h
ZG9iZTpkb2NpZDpwaG90b3Nob3A6OGRiMmM0OTUtMzc1Mi1iNTQ0LWI2YjEtNTQ1MmM3ZjkzMDVh
PC9yZGY6bGk+IDxyZGY6bGk+YWRvYmU6ZG9jaWQ6cGhvdG9zaG9wOmYwNGEwN2E3LWU0YWEtZDI0
NS05ZjExLWRiZjY3YzhiYWIyOTwvcmRmOmxpPiA8L3JkZjpCYWc+IDwvcGhvdG9zaG9wOkRvY3Vt
ZW50QW5jZXN0b3JzPiA8L3JkZjpEZXNjcmlwdGlvbj4gPC9yZGY6UkRGPiA8L3g6eG1wbWV0YT4g
PD94cGFja2V0IGVuZD0iciI/Ps9ll/cAAAVZSURBVHic7d3BTdxQGEbRTOT+IsgisIAGkFhQEhvE
AiR6CnWQDhIhP8X/cM8pwDIe++qt+E43Vxcf3wZ7fHo9HX0PdN1eX47+PlarfW/fj74BgKMIIJAl
gECWAAJZAghkCSCQJYBAlgACWQIIZAkgkCWAQJYAAlkCCGQJIJAlgECWAAJZAghkCSCQJYBA1vbj
56+lF3x7eV56vdVsPHCk2vc2nRMgkCWAQJYAAlkCCGQJIJAlgECWAAJZAghkCSCQJYBAlgACWQII
ZAkgkCWAQJYAAlkCCGQJIJAlgECWAAJZ29E38C/TNzymbzxMf36r2UDZZ/r7svr3dQIEsgQQyBJA
IEsAgSwBBLIEEMgSQCBLAIEsAQSyBBDIEkAgSwCBLAEEsgQQyBJAIEsAgSwBBLIEEMgSQCBr+SbI
6o0MZpm+gVIz/Xub/vs6AQJZAghkCSCQJYBAlgACWQIIZAkgkCWAQJYAAlkCCGQJIJAlgECWAAJZ
AghkCSCQJYBAlgACWQIIZAkgkLVN/5/9003fZOBr8/3u4wQIZAkgkCWAQJYAAlkCCGQJIJAlgECW
AAJZAghkCSCQJYBAlgACWQIIZAkgkCWAQJYAAlkCCGQJIJAlgEDW9vj0ejr6Jv6n2+vLj6Pv4W9s
jOwz/fdlFidAIEsAgSwBBLIEEMgSQCBLAIEsAQSyBBDIEkAgSwCBLAEEsgQQyBJAIEsAgSwBBLIE
EMgSQCBLAIEsAQSyTjdXFzYUdli94fH28rz0ejU2VWaZ/j47AQJZAghkCSCQJYBAlgACWQIIZAkg
kCWAQJYAAlkCCGQJIJAlgECWAAJZAghkCSCQJYBAlgACWQIIZAkgkLVN31CYvinA11Z7/1b3YHpf
nACBLAEEsgQQyBJAIEsAgSwBBLIEEMgSQCBLAIEsAQSyBBDIEkAgSwCBLAEEsgQQyBJAIEsAgSwB
BLIEEMjaVm8eTN8AmG7186ttWrBP7X1xAgSyBBDIEkAgSwCBLAEEsgQQyBJAIEsAgSwBBLIEEMgS
QCBLAIEsAQSyBBDIEkAgSwCBLAEEsgQQyBJAIGs7+gbOnU2Vfab/vdPvr7bhsZoTIJAlgECWAAJZ
AghkCSCQJYBAlgACWQIIZAkgkCWAQJYAAlkCCGQJIJAlgECWAAJZAghkCSCQJYBAlgACWeM3QaZv
Mkzn+c0yfcOj9r44AQJZAghkCSCQJYBAlgACWQIIZAkgkCWAQJYAAlkCCGQJIJAlgECWAAJZAghk
CSCQJYBAlgACWQIIZAkgkHW6ubr4OPomzlltQwE+Y/UGyurvzQkQyBJAIEsAgSwBBLIEEMgSQCBL
AIEsAQSyBBDIEkAgSwCBLAEEsgQQyBJAIEsAgSwBBLIEEMgSQCBLAIGsbfUFbWTA+Vq94bHa6vtz
AgSyBBDIEkAgSwCBLAEEsgQQyBJAIEsAgSwBBLIEEMgSQCBLAIEsAQSyBBDIEkAgSwCBLAEEsgQQ
yBJAIGv5JshqqzcApm+WrP57f7+/L73e3f3D0uut5vntU/s+nACBLAEEsgQQyBJAIEsAgSwBBLIE
EMgSQCBLAIEsAQSyBBDIEkAgSwCBLAEEsgQQyBJAIEsAgSwBBLIEEMjapm8A1KzeoKjx/PapbfA4
AQJZAghkCSCQJYBAlgACWQIIZAkgkCWAQJYAAlkCCGQJIJAlgECWAAJZAghkCSCQJYBAlgACWQII
ZAkgkLWt3gBgn7v7h6Nv4ax5fnyGEyCQJYBAlgACWQIIZAkgkCWAQJYAAlkCCGQJIJAlgECWAAJZ
AghkCSCQJYBAlgACWQIIZAkgkCWAQJYAAll/AL7FgoU6lGv/AAAAAElFTkSuQmCC
`
