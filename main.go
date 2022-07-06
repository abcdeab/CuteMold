package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/atotto/clipboard"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	SIZE_X = 400
	SIZE_Y = 250
	ZOOM   = 3

	MUTATE     = 50
	LEN_GENOME = 50

	MAX_GENOME = 5000
	MAX_MOLD   = 10000

	RIGHT = 0
	TOP   = 1
	LEFT  = 2

	SPORE = -1
	NONE  = -2

	NUM = LEN_GENOME * 3
	R   = LEN_GENOME*3 + 1
	G   = LEN_GENOME*3 + 2
	B   = LEN_GENOME*3 + 3

	ENERGY_DAY  = 10
	ENERGY_MOLD = 5000

	TIME_CELL = 200
)

var ENERGY_LIGHT = 20
var TIME = 0
var PAUSE = false

// position molds
var cells [SIZE_X][SIZE_Y]Cell
var molds [MAX_MOLD]Mold

// gen n is in the positions 3n, 3n+1, 3n+2
// 3n - left growth
// 3n+1 - top growth
// 3n+2 - rigrt growth
// no bottom growth
var genomes [MAX_GENOME][LEN_GENOME*3 + 4]int

// for load gen in clipboard
var mouse_gen [LEN_GENOME*3 + 4]int

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
			genomes[new_genome][R] = 10 + rand.Intn(130)
			genomes[new_genome][G] = 10 + rand.Intn(130)
			genomes[new_genome][B] = 10 + rand.Intn(130)

			// create new mold and cell
			molds[new_mold] = Mold{new_genome, ENERGY_LIGHT * 10, 1, rand.Intn(50)}
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
		molds[new_mold] = Mold{new_genome, ENERGY_LIGHT * 10, 1, rand.Intn(50)}
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
			// new color
			genomes[new_genome][R] = 10 + rand.Intn(130)
			genomes[new_genome][G] = 10 + rand.Intn(130)
			genomes[new_genome][B] = 10 + rand.Intn(130)
			// mutate genome (5 gens)
			if rand.Intn(MUTATE) == 0 {
				for i := 0; i < 10; i++ {
					genomes[new_genome][rand.Intn(LEN_GENOME*3)] = rand_gen()
				}
			}
			// create new mold
			molds[new_mold] = Mold{new_genome, cells[x][y].time, 1, rand.Intn(50)}
		} else {
			// create new mold
			genomes[last_genome][NUM]++
			molds[new_mold] = Mold{last_genome, cells[x][y].time, 1, rand.Intn(50)}
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
	fmt.Println("Problem neitherhood")
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
	}

	// on/off pause
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		PAUSE = !PAUSE
	}

	// energy light
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		ENERGY_LIGHT--
		fmt.Println(ENERGY_LIGHT)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		ENERGY_LIGHT++
		fmt.Println(ENERGY_LIGHT)
	}

	// вывод информации
	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		fmt.Println("time", TIME)
	}
}

func mouse_click() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// cursor position
		x, y := ebiten.CursorPosition()
		x = int(x / ZOOM)
		y = int(y / ZOOM)

		// copy genom or add new mold next cursor
		if cells[x][y].mold != 0 {
			save, _ := json.Marshal(genomes[molds[cells[x][y].mold].genome])
			clipboard.WriteAll(string(save)) //  text genome save in clipboard
			fmt.Println("genome saved")
		} else {
			save, _ := clipboard.ReadAll()
			err := json.Unmarshal([]byte(save), &mouse_gen) // text genome load in mouse_gen
			if err != nil {
				// no panic
				fmt.Println("problem clickboard (no panic)")
			} else {
				load_genom(x, y)
				fmt.Println("genome loaded")
			}
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
	// screen output
	screen.ReplacePixels(g.pixels)
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

	// game run
	game := &Game{}
	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
