// version 2
package main

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"math/rand"
	"runtime"
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
	// developer mode
	DEVELOPER = false

	// size window
	WIN_X  = 1260
	WIN_Y  = 660
	SIZE_X = int(WIN_X / ZOOM_MIM)
	SIZE_Y = int(WIN_Y / ZOOM_MIM)

	// zoom
	ZOOM_MIM = 2
	ZOOM_MAX = 6

	// chance of mutation
	MUTATE = 50
	// genomes patameters
	LEN_GENOME = 300
	// places of special positions in the genome
	NUM = LEN_GENOME * 3
	R   = LEN_GENOME*3 + 1
	G   = LEN_GENOME*3 + 2
	B   = LEN_GENOME*3 + 3
	// direction
	RIGHT = 0
	TOP   = 1
	LEFT  = 2
	// specific genes
	SPORE = -1
	NONE  = -2

	// limiting the amount of mold
	MAX_GENOME = 2000
	MAX_MOLD   = 50000

	// light patameners
	ENERGY_LIGHT_MAX = 20
	ENERGY_DAY       = 5

	// cell aging
	TIME_CELL  = 200
	TIME_SPORE = 100
)

// zoom
var zoom = 2
var zoom_x = 0
var zoom_y = 0

// camera
var camera_flag = false
var camera_x, camera_y int

// time and pause
var world_time = 0
var pause = false

// light parameter
var energy_light = 18
var energy_visual_time = 300 // for draw

// for load gen in clipboard
var mouse_gen [LEN_GENOME*3 + 4]int

// text for draw
var NormalFont font.Face
var text_srting = "Press Q/W to increase/decrease the light."
var text_time = 300
var starting_text = true

//  molds and their positions
var cells [SIZE_X][SIZE_Y]Cell
var molds [MAX_MOLD]Mold

// gen n is in the positions 3n, 3n+1, 3n+2
// 3n - left growth
// 3n+1 - top growth
// 3n+2 - rigrt growth
// no bottom growth
var genomes [MAX_GENOME][LEN_GENOME*3 + 4]int

type Cell struct {
	mold   int // mold number
	n      int // active gene
	dx, dy int // direction of growth
	time   int // cell lifetime 0+
}

type Mold struct {
	genome int // genome number
	energy int // amount of energy
	num    int // amount of cell
	color  int // genome color correction
}

type Game struct {
	pixels []byte
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// x mod SIZE_X
func mod_x(x int) int {
	if x >= SIZE_X {
		return x - SIZE_X
	}
	if x < 0 {
		return x + SIZE_X
	}
	return x
}

// y mod SIZE_Y
func mod_y(y int) int {
	if y >= SIZE_Y {
		return y - SIZE_Y
	}
	if y < 0 {
		return y + SIZE_Y
	}
	return y
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
		text_time = world_time
		text_srting = "Panic! Too much genome!"
		//fmt.Println("Panic! Too much genome!")

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
		text_time = world_time
		text_srting = "Panic! Too much mold!"
		//fmt.Println("Panic! Too much mold!")
	}
	return new_mold
}

func delete_all() {
	// delete all genomes
	for i := 0; i < MAX_GENOME; i++ {
		genomes[i][NUM] = 0
	}
	// delete all molds
	for i := 0; i < MAX_MOLD; i++ {
		molds[i].num = 0
	}
	// delete all cells
	for i := 0; i < SIZE_X; i++ {
		for j := 0; j < SIZE_Y; j++ {
			cells[i][j].mold = 0
		}
	}
}

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
			molds[new_mold] = Mold{new_genome, energy_light, 1, rand.Intn(60)}
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
		molds[new_mold] = Mold{new_genome, energy_light, 1, rand.Intn(60)}
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
			for i := 0; i < LEN_GENOME*3; i++ {
				genomes[new_genome][i] = genomes[last_genome][i]
			}
			genomes[new_genome][NUM] = 1
			// new color for genome
			rand_color(new_genome)
			// mutate
			genomes[new_genome][rand.Intn(LEN_GENOME*3)] = rand_gen()

			// create new mold (new genome)
			molds[new_mold] = Mold{new_genome, 0, 1, rand.Intn(60)}
		} else {
			// create new mold (old genome)
			genomes[last_genome][NUM]++
			molds[new_mold] = Mold{last_genome, 0, 1, rand.Intn(60)}
		}

		// creade cell
		cells[x][y].mold = new_mold
		cells[x][y].n = 0
		cells[x][y].time = 0
	}
}

func add_cell(x, y, x2, y2, n int) {
	if cells[x2][y2].mold == 0 {
		// mold add num
		molds[cells[x][y].mold].num++

		// add cell
		cells[x2][y2].mold = cells[x][y].mold
		cells[x2][y2].n = n
		cells[x2][y2].time = 0

		// direction cell x
		cells[x2][y2].dx = mod_x(x2-x+1) - 1
		cells[x2][y2].dy = mod_y(y2-y+1) - 1
	}
}

func neighbor(x, y, dx, dy, dir int) (int, int) {
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
	text_time = world_time
	text_srting = "Panic! I can't find my neighbor!"
	// fmt.Println("Panic! I can't find my neighbor!")
	return 0, 0
}

func growth_cell(x, y int) {
	if cells[x][y].n != SPORE {
		// growth cell on four direction
		for dir := 0; dir < 3; dir++ {
			// new cell gen in the genome
			next_n := genomes[molds[cells[x][y].mold].genome][cells[x][y].n*3+dir]
			if next_n != NONE {
				x2, y2 := neighbor(x, y, cells[x][y].dx, cells[x][y].dy, dir)
				add_cell(x, y, x2, y2, next_n)
			}
		}
	}
}

func add_energy(x, y int) {
	// potential mold, which will get the energy
	m := max(max(cells[mod_x(x+1)][y].mold, cells[mod_x(x-1)][y].mold), max(cells[x][mod_y(y+1)].mold, cells[x][mod_y(y-1)].mold))

	// mold gets energy if there are no other molds near the cell
	if m != 0 {
		if cells[mod_x(x+1)][y].mold == m || cells[mod_x(x+1)][y].mold == 0 {
			if cells[mod_x(x-1)][y].mold == m || cells[mod_x(x-1)][y].mold == 0 {
				if cells[x][mod_y(y+1)].mold == m || cells[x][mod_y(y+1)].mold == 0 {
					if cells[x][mod_y(y-1)].mold == m || cells[x][mod_y(y-1)].mold == 0 {
						molds[m].energy += energy_light
					}
				}
			}
		}
	}
}

func delete_cell(x, y int) {
	delete_mold := cells[x][y].mold
	delete_genome := molds[delete_mold].genome

	// delete cell
	if cells[x][y].n == SPORE && cells[x][y].time >= TIME_SPORE {
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
	if !pause {
		// photosynthesis, food and aging
		for x := 0; x < SIZE_X; x++ {
			for y := 0; y < SIZE_Y; y++ {
				if cells[x][y].mold != 0 {
					molds[cells[x][y].mold].energy -= ENERGY_DAY * (1 + int(cells[x][y].time/TIME_CELL))
					cells[x][y].time++
				} else {
					add_energy(x, y)
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
		starting_text = false
	}

	// on/off pause
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		pause = !pause
	}

	// energy light
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		energy_light--
		energy_light = max(0, energy_light)
		energy_visual_time = world_time
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		energy_light++
		energy_light = min(energy_light, ENERGY_LIGHT_MAX)
		energy_visual_time = world_time
	}

	// print info
	if inpututil.IsKeyJustPressed(ebiten.KeyI) && DEVELOPER {
		num_g := 0
		for i := 0; i < MAX_GENOME; i++ {
			if genomes[i][NUM] != 0 {
				num_g++
			}
		}
		num_m := 0
		for i := 0; i < MAX_MOLD; i++ {
			if molds[i].num != 0 {
				num_m++
			}
		}
		fmt.Println("time", world_time, "genome", num_g, "mold", num_m)
	}

	// print fps
	if inpututil.IsKeyJustPressed(ebiten.KeyJ) && DEVELOPER {
		fmt.Println(ebiten.CurrentFPS())
	}

	// delete all
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		delete_all()
		text_time = world_time
		text_srting = "Delete all molds."
	}

	// toggle fullscreen
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	// toggle fullscreen
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		ebiten.SetFullscreen(false)
	}
}

func mouse_click() {
	// left mouse
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// cursor position
		x, y := ebiten.CursorPosition()
		if 0 <= x && x < WIN_X && 0 <= y && y < WIN_Y {
			// world position
			x = mod_x(zoom_x + int(x/zoom))
			y = mod_y(zoom_y + int(y/zoom))
			if cells[x][y].mold != 0 {
				// text genome save in clipboard
				save, _ := json.Marshal(genomes[molds[cells[x][y].mold].genome])
				clipboard.WriteAll(string(save))
				//fmt.Println("Genome saved in clipboard.")
				text_time = world_time
				text_srting = "Genome saved in clipboard."
				// mold parameters
				if DEVELOPER {
					fmt.Println("x", x, "y", y, "gen", cells[x][y].n, "age", cells[x][y].time, "energy", molds[cells[x][y].mold].energy, "mold", cells[x][y].mold)
				}
			} else {
				// text genome load in mouse_gen
				save, _ := clipboard.ReadAll()
				if len(save) < 500 {
					// сonversion of the old version genome (50 genes) into the new version genome (500 genes)
					var last_mouse_gen [154]int
					err := json.Unmarshal([]byte(save), &last_mouse_gen)
					if err == nil {
						for i := 0; i < 150; i++ {
							mouse_gen[i] = last_mouse_gen[i]
						}
						for i := 154; i < LEN_GENOME*3; i++ {
							mouse_gen[i] = rand_gen()
						}
						mouse_gen[NUM] = 1
						mouse_gen[R] = last_mouse_gen[151]
						mouse_gen[G] = last_mouse_gen[152]
						mouse_gen[B] = last_mouse_gen[153]

						load_genom(x, y)
						text_time = world_time
						text_srting = "Genome (old version) loaded from clipboard."
						starting_text = false
					} else {
						// no panic
						text_time = world_time
						text_srting = "Can't load: no genome on the clipboard."
					}
				} else {
					// loading the new version genome
					err := json.Unmarshal([]byte(save), &mouse_gen)
					if err == nil {
						load_genom(x, y)
						text_time = world_time
						text_srting = "Genome loaded from clipboard."
						starting_text = false
					} else {
						// no panic
						text_time = world_time
						text_srting = "Can't load: no genome on the clipboard."
					}
				}
			}
		} else {
			text_time = world_time
			text_srting = "Click out of the world."
		}
	}

	// right mouse
	// change zoom
	if !camera_flag {
		_, z := ebiten.Wheel()
		x, y := ebiten.CursorPosition()
		zoom_x = mod_x(zoom_x + int(x/zoom))
		zoom_y = mod_y(zoom_y + int(y/zoom))

		zoom += int(z)
		zoom = max(zoom, ZOOM_MIM)
		zoom = min(zoom, ZOOM_MAX)

		zoom_x = mod_x(zoom_x - int(x/zoom))
		zoom_y = mod_y(zoom_y - int(y/zoom))
	}

	// change cameta posion
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		camera_flag = true
		x, y := ebiten.CursorPosition()
		camera_x = mod_x(zoom_x + int(x/zoom))
		camera_y = mod_y(zoom_y + int(y/zoom))
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		x, y := ebiten.CursorPosition()
		zoom_x = mod_x(camera_x - int(x/zoom))
		zoom_y = mod_y(camera_y - int(y/zoom))
	} else {
		camera_flag = false
	}
}

func (g *Game) Update() error {
	// mouse and keyboard
	mouse_click()
	key_press()
	// update
	if runtime.NumGoroutine() <= 2 {
		go update()
		world_time++
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// create pixels
	if g.pixels == nil {
		g.pixels = make([]byte, WIN_X*WIN_Y*4)
	}

	// draw cells
	for x := 0; x < int(WIN_X/zoom); x++ {
		for y := 0; y < int(WIN_Y/zoom); y++ {
			x0 := mod_x(x + zoom_x)
			y0 := mod_y(y + zoom_y)

			if cells[x0][y0].mold != 0 {
				// draw cell
				color_r := byte(genomes[molds[cells[x0][y0].mold].genome][R] + molds[cells[x0][y0].mold].color)
				color_g := byte(genomes[molds[cells[x0][y0].mold].genome][G] + molds[cells[x0][y0].mold].color)
				color_b := byte(genomes[molds[cells[x0][y0].mold].genome][B] + molds[cells[x0][y0].mold].color)
				for i := 0; i < zoom; i++ {
					for j := 0; j < zoom; j++ {
						pic := ((y*zoom+j)*WIN_X + x*zoom + i) * 4
						g.pixels[pic] = color_r
						g.pixels[pic+1] = color_g
						g.pixels[pic+2] = color_b
						g.pixels[pic+3] = 0xff
					}
				}
				// draw spore
				if cells[x0][y0].n == SPORE {
					for i := 1; i < zoom-1; i++ {
						for j := 1; j < zoom-1; j++ {
							pic := ((y*zoom+j)*WIN_X + x*zoom + i) * 4
							g.pixels[pic] = 0
							g.pixels[pic+1] = 0
							g.pixels[pic+2] = 0
							g.pixels[pic+3] = 0xff
						}
					}
				}

			} else {
				// draw nil cell
				for i := 0; i < zoom; i++ {
					for j := 0; j < zoom; j++ {
						pic := ((y*zoom+j)*WIN_X + x*zoom + i) * 4
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
	if world_time-energy_visual_time < 120 {
		for x := 20; x < 20+energy_light*16; x++ {
			for y := 20; y < 40; y++ {
				pic := (y*WIN_X + x) * 4
				g.pixels[pic] = 0xc0
				g.pixels[pic+1] = 0xc3
				g.pixels[pic+2] = 0x50
				g.pixels[pic+3] = 0xff
			}
		}
		for x := 20 + energy_light*16; x < 20+ENERGY_LIGHT_MAX*16; x++ {
			for y := 20; y < 40; y++ {
				pic := (y*WIN_X + x) * 4
				g.pixels[pic] = 0x60
				g.pixels[pic+1] = 0x60
				g.pixels[pic+2] = 0x60
				g.pixels[pic+3] = 0xff
			}
		}
	}

	// screen output
	screen.ReplacePixels(g.pixels)

	// graw start hello
	if starting_text {
		text.Draw(screen, "Press G to generate new molds", NormalFont, int(WIN_X/2)-150, int(WIN_Y/2), color.White)
	}

	// graw text light
	if world_time-energy_visual_time < 120 {
		text.Draw(screen, fmt.Sprint(energy_light), NormalFont, 20+ENERGY_LIGHT_MAX*8, 37, color.Black)
	}

	// graw copy/load
	if world_time-text_time < 120 {
		text.Draw(screen, text_srting, NormalFont, 20, 60, color.White)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	// size window
	return WIN_X, WIN_Y
}

func main() {
	// seed for random
	rand.Seed(time.Now().UnixNano())

	// create window
	ebiten.SetWindowSize(WIN_X, WIN_Y)
	ebiten.SetWindowTitle("Cute mold v2")
	//ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)

	// logo
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
IjIwMjItMDctMDdUMTQ6MDQ6MDcrMDM6MDAiIHhtcDpNb2RpZnlEYXRlPSIyMDIyLTA3LTA3VDE0
OjA0OjA3KzAzOjAwIiB4bXBNTTpJbnN0YW5jZUlEPSJ4bXAuaWlkOmFkYjY0Nzg4LThmYzYtNTA0
My1iYjM0LWZmMGE3MWM0NmFlZCIgeG1wTU06RG9jdW1lbnRJRD0iYWRvYmU6ZG9jaWQ6cGhvdG9z
aG9wOjZkZDRjMjY4LTdmZWUtYmI0Yi04YjliLTczODk0ZWI5OTc0ZSIgeG1wTU06T3JpZ2luYWxE
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
PHJkZjpsaSBzdEV2dDphY3Rpb249InNhdmVkIiBzdEV2dDppbnN0YW5jZUlEPSJ4bXAuaWlkOmFk
YjY0Nzg4LThmYzYtNTA0My1iYjM0LWZmMGE3MWM0NmFlZCIgc3RFdnQ6d2hlbj0iMjAyMi0wNy0w
N1QxNDowNDowNyswMzowMCIgc3RFdnQ6c29mdHdhcmVBZ2VudD0iQWRvYmUgUGhvdG9zaG9wIEND
IDIwMTkgKFdpbmRvd3MpIiBzdEV2dDpjaGFuZ2VkPSIvIi8+IDwvcmRmOlNlcT4gPC94bXBNTTpI
aXN0b3J5PiA8cGhvdG9zaG9wOkRvY3VtZW50QW5jZXN0b3JzPiA8cmRmOkJhZz4gPHJkZjpsaT5h
ZG9iZTpkb2NpZDpwaG90b3Nob3A6OGRiMmM0OTUtMzc1Mi1iNTQ0LWI2YjEtNTQ1MmM3ZjkzMDVh
PC9yZGY6bGk+IDxyZGY6bGk+YWRvYmU6ZG9jaWQ6cGhvdG9zaG9wOmYwNGEwN2E3LWU0YWEtZDI0
NS05ZjExLWRiZjY3YzhiYWIyOTwvcmRmOmxpPiA8L3JkZjpCYWc+IDwvcGhvdG9zaG9wOkRvY3Vt
ZW50QW5jZXN0b3JzPiA8L3JkZjpEZXNjcmlwdGlvbj4gPC9yZGY6UkRGPiA8L3g6eG1wbWV0YT4g
PD94cGFja2V0IGVuZD0iciI/PmcsnWMAAAVSSURBVHic7d3BbdRQGEZRD3JliVIIiLKiZIFYpBMW
SLCjnNABJPIb/A/3nAIsx2NfvVW+y/3D3es22+XsGyBt+vexWup7+3D2DQCcRQCBLAEEsgQQyBJA
IEsAgSwBBLIEEMgSQCBLAIEsAQSyBBDIEkAgSwCBLAEEsgQQyBJAIEsAgSwBBLIuX1++LN08eHp8
Xnm5bVu/UWDjgfdY+r58+vxx5eVu4XsbzQkQyBJAIEsAgSwBBLIEEMgSQCBLAIEsAQSyBBDIEkAg
SwCBLAEEsgQQyBJAIEsAgSwBBLIEEMgSQCBLAIGs/ewbeIPRGx43sPEw+vldQWrT4gqmvy9Lf18n
QCBLAIEsAQSyBBDIEkAgSwCBLAEEsgQQyBJAIEsAgSwBBLIEEMgSQCBLAIEsAQSyBBDIEkAgSwCB
LAEEspZvgqzeyGCWG9hASZn+vU3/fZ0AgSwBBLIEEMgSQCBLAIEsAQSyBBDIEkAgSwCBLAEEsgQQ
yBJAIEsAgSwBBLIEEMgSQCBLAIEsAQSyBBDI2qf/z/7ppm8y8H/z/R7jBAhkCSCQJYBAlgACWQII
ZAkgkCWAQJYAAlkCCGQJIJAlgECWAAJZAghkCSCQJYBAlgACWQIIZAkgkCWAQNa+bdvl7Jv4x17P
voE/sTFy2Ojfl1mcAIEsAQSyBBDIEkAgSwCBLAEEsgQQyBJAIEsAgSwBBLIEEMgSQCBLAIEsAQSy
BBDIEkAgSwCBLAEEsgQQyNo3GwqjPD0+n30LN622qTL9773C+7y0V06AQJYAAlkCCGQJIJAlgECW
AAJZAghkCSCQJYBAlgACWQIIZAkgkCWAQJYAAlkCCGQJIJAlgECWAAJZAghk7cFNAXiz2vu3ugfT
++IECGQJIJAlgECWAAJZAghkCSCQJYBAlgACWQIIZAkgkCWAQJYAAlkCCGQJIJAlgECWAAJZAghk
CSCQJYBA1r5682D6BsB0q59fbdOCY2rvixMgkCWAQJYAAlkCCGQJIJAlgECWAAJZAghkCSCQJYBA
lgACWQIIZAkgkCWAQJYAAlkCCGQJIJAlgECWAAJZ+9k3cOtsqhwz/e+dfn+1DY/VnACBLAEEsgQQ
yBJAIEsAgSwBBLIEEMgSQCBLAIEsAQSyBBDIEkAgSwCBLAEEsgQQyBJAIEsAgSwBBLIEEMi63D/c
va684PQNBTjT9A2P2vfrBAhkCSCQJYBAlgACWQIIZAkgkCWAQJYAAlkCCGQJIJAlgECWAAJZAghk
CSCQJYBAlgACWQIIZAkgkCWAQNbyTZCa2oYCvMfqDZTV35sTIJAlgECWAAJZAghkCSCQJYBAlgAC
WQIIZAkgkCWAQJYAAlkCCGQJIJAlgECWAAJZAghkCSCQJYBAlgACWcs3QWxkwO1aveExnRMgkCWA
QJYAAlkCCGQJIJAlgECWAAJZAghkCSCQJYBAlgACWQIIZAkgkCWAQJYAAlkCCGQJIJAlgECWAAJZ
+9k38DerNwqmb5b8+P5t7fV+/lp6Pc/vmOnPb/r9re6BEyCQJYBAlgACWQIIZAkgkCWAQJYAAlkC
CGQJIJAlgECWAAJZAghkCSCQJYBAlgACWQIIZAkgkCWAQJYAAln79A2AmtUbFDWe3zG1DR4nQCBL
AIEsAQSyBBDIEkAgSwCBLAEEsgQQyBJAIEsAgSwBBLIEEMgSQCBLAIEsAQSyBBDIEkAgSwCBLAEE
svbVGwAcM31DYTrPj/dwAgSyBBDIEkAgSwCBLAEEsgQQyBJAIEsAgSwBBLIEEMgSQCBLAIEsAQSy
BBDIEkAgSwCBLAEEsgQQyBJAIOs3EnlyUx/dweQAAAAASUVORK5CYII=
`
