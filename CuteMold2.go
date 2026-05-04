// version 2
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/atotto/clipboard"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

const (
	DEVELOPER = false
	TRANSLATE = false
	
	SIZE_X = 4096 
	SIZE_Y = 4096
	MASK_X = SIZE_X - 1
	MASK_Y = SIZE_Y - 1

	ZOOM_MIM = 1
	ZOOM_MAX = 20

	MUTATE = 100
	
	LEN_GENOME = 512
	LEN_GROWTH = 128
	
	RIGHT = 0
	TOP   = 1
	LEFT  = 2
	
	TYPE  = 3
	
	SPORE_N = 0
	SPORE_K = 30

	MAX_MOLD   = 500000

	ENERGY_DAY       =  1
	ENERGY_SPORE     =  8
	ENERGY_LIGHT_MAX = 16
	
	TIME_CELL  =   20
	TIME_SPORE =   60
	
	TIME_SHOW_NOTICE = 120
	
	vowels     = "aeiouy"
	consonants = "bcdfghjklmnprstvwxz"
)

var NUM_WORKERS int
var isUpdating int32
var isSweeping int32
var spawnMu sync.Mutex

type Game struct {
	pixels []byte
	moldsScreen *ebiten.Image
}

var WIN_X  = 1800
var WIN_Y  = 900

var camera_flag = false
var camera_x, camera_y int
var zoom = 2
var zoom_x = 0
var zoom_y = 0

var world_time = 0
var pause = false
var is_saving = false

var show_world = true
var show_inform_control = true
var show_inform_technical = false
var show_light_map = true
var show_starting_text = true

var NormalFont font.Face
var text_string string
var text_time int

var controlMenuImage *ebiten.Image

func drawTextWithOutline(screen *ebiten.Image, msg string, f font.Face, x, y int, c color.Color) {
	text.Draw(screen, msg, f, x+1, y+1, color.Black)
	text.Draw(screen, msg, f, x-1, y-1, color.Black)
	text.Draw(screen, msg, f, x+1, y-1, color.Black)
	text.Draw(screen, msg, f, x-1, y+1, color.Black)
	
	text.Draw(screen, msg, f, x, y, c)
}

func calc_nodes_text_width() {
	for i := range nodes {
		bounds := text.BoundString(NormalFont, nodes[i].Name)
		nodes[i].TextWidth = bounds.Dx()
	}
}

func create_control_menu_image() {
	controlMenuImage = ebiten.NewImage(WIN_X, WIN_Y)
	slip := 40
	
	if TRANSLATE {
		drawTextWithOutline(controlMenuImage, "G - сгенерировать случайные плесени", NormalFont, 20, slip, color.White)
		slip += 20
		drawTextWithOutline(controlMenuImage, "Клик колеса мыши - убить плесени в радиусе 50", NormalFont, 20, slip, color.White)
		slip += 30
		drawTextWithOutline(controlMenuImage, "P - вкл/выкл паузу", NormalFont, 20, slip, color.White)
		slip += 20			
		drawTextWithOutline(controlMenuImage, "S - сохранить мир", NormalFont, 20, slip, color.White)
		slip += 20
		drawTextWithOutline(controlMenuImage, "L - загрузить мир", NormalFont, 20, slip, color.White)
		slip += 20
		drawTextWithOutline(controlMenuImage, "E - сохранить мир как большую картинку", NormalFont, 20, slip, color.White)
		slip += 20		
		drawTextWithOutline(controlMenuImage, "D - полностью удалить мир", NormalFont, 20, slip, color.White)
		slip += 30
		drawTextWithOutline(controlMenuImage, "Левый клик мыши - скопировать/вставить геном в буфер обмена", NormalFont, 20, slip, color.White)
		slip += 20
		drawTextWithOutline(controlMenuImage, "Правый клик мыши - перемещение камеры", NormalFont, 20, slip, color.White)
		slip += 20
		drawTextWithOutline(controlMenuImage, "Колесо мыши - приблизить/отдалить", NormalFont, 20, slip, color.White)
		slip += 30
		drawTextWithOutline(controlMenuImage, "J - показать параметр освещённости", NormalFont, 20, slip, color.White)
		slip += 20
		drawTextWithOutline(controlMenuImage, "I - скрыть этот информационный блок", NormalFont, 20, slip, color.White)
	} else {
		drawTextWithOutline(controlMenuImage, "G - generate random molds", NormalFont, 20, slip, color.White)
		slip += 20
		drawTextWithOutline(controlMenuImage, "Mouse Wheel click - kill molds within a 50 radius", NormalFont, 20, slip, color.White)
		slip += 30
		drawTextWithOutline(controlMenuImage, "P - turn on/off pause", NormalFont, 20, slip, color.White)
		slip += 20			
		drawTextWithOutline(controlMenuImage, "S - save world", NormalFont, 20, slip, color.White)
		slip += 20
		drawTextWithOutline(controlMenuImage, "L - load world", NormalFont, 20, slip, color.White)
		slip += 20
		drawTextWithOutline(controlMenuImage, "E - export world as a huge image", NormalFont, 20, slip, color.White)
		slip += 20
		drawTextWithOutline(controlMenuImage, "D - delete the world completely", NormalFont, 20, slip, color.White)
		slip += 30
		drawTextWithOutline(controlMenuImage, "Left Click - copy/paste genome to clipboard", NormalFont, 20, slip, color.White)
		slip += 20
		drawTextWithOutline(controlMenuImage, "Right Click - pan camera", NormalFont, 20, slip, color.White)
		slip += 20
		drawTextWithOutline(controlMenuImage, "Mouse Wheel - zoom screen", NormalFont, 20, slip, color.White)
		slip += 30
		drawTextWithOutline(controlMenuImage, "J - show illumination parameter", NormalFont, 20, slip, color.White)
		slip += 20
		drawTextWithOutline(controlMenuImage, "I - hide this information block", NormalFont, 20, slip, color.White)
		slip += 40
	}
}

var energy_light = 16

var lightMapImage *ebiten.Image

type LightNode struct {
	Name  string
	X, Y  int 
	Power int
	TextWidth int
}
var nodes []LightNode

func update_lightmap_image() {
	if lightMapImage == nil {
		lightMapImage = ebiten.NewImage(SIZE_X, SIZE_Y)
	}

	pixels := make([]byte, SIZE_X*SIZE_Y*4)

	for x := 0; x < SIZE_X; x++ {
		for y := 0; y < SIZE_Y; y++ {
			idx := x*SIZE_Y + y
			
			local_light := min(int(energy_light), int(light_map[idx]))
			color_val := local_light * 8
			if color_val > 255 { color_val = 255 }
			bg_c := byte(color_val)
			
			pIdx := (y*SIZE_X + x) * 4
			pixels[pIdx]   = bg_c
			pixels[pIdx+1] = bg_c
			pixels[pIdx+2] = bg_c
			pixels[pIdx+3] = 0xff
		}
	}
	
	lightMapImage.ReplacePixels(pixels)
}



type Bitset [2048]byte

func (d *Bitset) Set(index int, value byte) {
	d[index] = value & 255
}

func (d *Bitset) Get(index int) byte {
	return d[index]
}

func (d *Bitset) RandomizeHalfZeros() {
	rand.Read(d[:])
}

func (d *Bitset) MutateOne() {
	index := rand.Intn(2048)
	d[index] = byte(rand.Intn(256))
}

func rand_color() (byte, byte, byte) {
	return byte(20 + rand.Intn(130)), byte(20 + rand.Intn(130)), byte(20 + rand.Intn(130))
}

func generate_name() [6]byte {
	var name [6]byte
	for i := 0; i < 6; i += 2 {
		name[i] = consonants[rand.Intn(len(consonants))]
		name[i+1] = vowels[rand.Intn(len(vowels))]
	}
	name[0] -= 32 
	return name
}


var mouse_gen Bitset

var alphabet256 = []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzАБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯабвгдеёжзийклмнопрстуфхцчшщъыьэюяÀÁÂÃÄÅÆÇÈÉÊËÌÍÎÏÐÑÒÓÔÕÖØÙÚÛÜÝÞßàáâãäåæçèéêëìíîïðñòóôõöøùúûüýþÿΑΒΓΔΕΖΗΘΙΚΛΜΝΞΟΠΡΣΤΥΦΧΨΩαβγδεζηθικλμνξοπρστυφχψωĄąĆćĘęŁłŃńŚśŹźŻżŠš")
var revMap256 map[rune]byte

func encodeBase256(data []byte) string {
	result := make([]rune, len(data))
	for i, b := range data {
		result[i] = alphabet256[b]
	}
	return string(result)
}

func decodeBase256(s string) []byte {
	result := make([]byte, 0, len(s))
	for _, r := range s {
		if val, ok := revMap256[r]; ok {
			result = append(result, val)
		}
	}
	return result
}

func compress_genome(gen *Bitset) string {
	var buf bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	gz.Write(gen[:]) 
	gz.Close() 
	
	return encodeBase256(buf.Bytes())
}

func decompress_genome(data string, gen *Bitset) error {
	rawBytes := decodeBase256(data)
	
	gz, err := gzip.NewReader(bytes.NewReader(rawBytes))
	if err != nil {
		return err
	}
	defer gz.Close()
	
	_, err = io.ReadFull(gz, gen[:])
	return err
}



type Cell struct {
	mold uint32
	time uint16
	n    int16
	dx   int8
	dy   int8
	spore bool
}

var cells []Cell

var light_map []int8

var cavity_map []uint32
var temp_energies []int64

func getCellIdx(x, y int) int {
	return x*SIZE_Y + y
}

func mod_x(x int) int {
	return x & MASK_X
}
func mod_y(y int) int {
	return y & MASK_Y
}

func sweep_cavities_background() {
	for i := 0; i < MAX_MOLD; i++ {
		temp_energies[i] = 0
	}

	bgWorkers := 2
	var wg sync.WaitGroup
	
	chunkSizeX := SIZE_X / bgWorkers
	for w := 0; w < bgWorkers; w++ {
		startX := w * chunkSizeX
		endX := (w + 1) * chunkSizeX
		if w == bgWorkers-1 { endX = SIZE_X }
		
		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			for x := s; x < e; x++ {
				base := x * SIZE_Y
				firstMoldIdx := -1
				for y := 0; y < SIZE_Y; y++ {
					if cells[base+y].mold != 0 {
						firstMoldIdx = y; break
					}
				}
				if firstMoldIdx != -1 {
					current_mold := cells[base+firstMoldIdx].mold
					empty_start_offset := -1
					for offset := 0; offset <= SIZE_Y; offset++ {
						y := (firstMoldIdx + offset) & MASK_Y 
						idx := base + y
						m := cells[idx].mold
						
						if m != 0 {
							if empty_start_offset != -1 {
								if current_mold == m {
									for e_off := empty_start_offset; e_off < offset; e_off++ {
										eidx := base + ((firstMoldIdx + e_off) & MASK_Y)
										cavity_map[eidx] = current_mold
									}
								} else {
									for e_off := empty_start_offset; e_off < offset; e_off++ {
										eidx := base + ((firstMoldIdx + e_off) & MASK_Y)
										cavity_map[eidx] = 0
									}
								}
								empty_start_offset = -1
							}
							current_mold = m
						} else {
							if empty_start_offset == -1 { empty_start_offset = offset }
						}
					}
				} else {
					for y := 0; y < SIZE_Y; y++ { cavity_map[base+y] = 0 }
				}
			}
		}(startX, endX)
	}
	wg.Wait()

	chunkSizeY := SIZE_Y / bgWorkers
	for w := 0; w < bgWorkers; w++ {
		startY := w * chunkSizeY
		endY := (w + 1) * chunkSizeY
		if w == bgWorkers-1 { endY = SIZE_Y }
		
		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			for y := s; y < e; y++ {
				firstMoldIdx := -1
				for x := 0; x < SIZE_X; x++ {
					if cells[x*SIZE_Y+y].mold != 0 {
						firstMoldIdx = x; break
					}
				}
				if firstMoldIdx != -1 {
					current_mold := cells[firstMoldIdx*SIZE_Y+y].mold
					empty_start_offset := -1
					
					for offset := 0; offset <= SIZE_X; offset++ {
						x := (firstMoldIdx + offset) & MASK_X 
						idx := x*SIZE_Y + y
						m := cells[idx].mold
						
						if m != 0 {
							if empty_start_offset != -1 {
								if current_mold == m {
									for e_off := empty_start_offset; e_off < offset; e_off++ {
										ex := (firstMoldIdx + e_off) & MASK_X 
										eidx := ex*SIZE_Y + y
										
										if cavity_map[eidx] == current_mold {
											local_light := min(int(energy_light), int(light_map[eidx]))
											if local_light > 0 {
												atomic.AddInt64(&temp_energies[current_mold], int64(local_light * 2))
											}
										} else {
											cavity_map[eidx] = 0 
										}
									}
								} else {
									for e_off := empty_start_offset; e_off < offset; e_off++ {
										ex := (firstMoldIdx + e_off) & MASK_X
										cavity_map[ex*SIZE_Y + y] = 0
									}
								}
								empty_start_offset = -1
							}
							current_mold = m
						} else {
							if empty_start_offset == -1 { empty_start_offset = offset }
						}
					}
				} else {
					for x := 0; x < SIZE_X; x++ { cavity_map[x*SIZE_Y+y] = 0 }
				}
			}
		}(startY, endY)
	}
	wg.Wait()

	for i := 1; i < MAX_MOLD; i++ {
		if molds[i].leave {
			atomic.StoreInt64(&molds[i].cavity_energy, temp_energies[i])
		}
	}

	atomic.StoreInt32(&isSweeping, 0)
}


type Mold struct {
	name [6]byte
	genome Bitset
	energy int64
	leave bool
	free bool
	r, g, b byte
	rc, gc, bc byte
	cavity_energy int64
}
var molds [MAX_MOLD]Mold

var workers_consumption [][]int64


func init() {
	rand.Seed(time.Now().UnixNano())
	
	NUM_WORKERS = runtime.NumCPU() - 2
	if NUM_WORKERS < 1 {
		NUM_WORKERS = 1
	}
	if DEVELOPER {
		fmt.Println("Потоков: ", NUM_WORKERS)
	}
	
	win_size_x, win_size_y := ebiten.ScreenSizeInFullscreen()
	WIN_X = min(WIN_X, win_size_x - 100)
	WIN_Y = min(WIN_Y, win_size_y - 100)
	if DEVELOPER { 
		fmt.Println("Размеры экрана: ", win_size_x, win_size_y)
		fmt.Println("Размеры окна: ", WIN_X, WIN_Y)
	}
	
	if TRANSLATE {
		text_string = "Программа запущена."
	} else {
		text_string = "The program has started."
	}
	text_time = 300

	if DEVELOPER { show_inform_control = false}
	
	for i := 0; i < MAX_MOLD; i++ {
		molds[i].free = true
		molds[i].leave = false
	}
	
	cells = make([]Cell, SIZE_X*SIZE_Y)
	for i := 0; i < SIZE_X*SIZE_Y; i++ {
		cells[i].mold = 0
	}
	
	cavity_map = make([]uint32, SIZE_X*SIZE_Y)
	temp_energies = make([]int64, MAX_MOLD)
	
	generate_random_nodes()
	generate_light_map()	
	
	revMap256 = make(map[rune]byte)
	for i, r := range alphabet256 {
		revMap256[r] = byte(i)
	}
	
	workers_consumption = make([][]int64, NUM_WORKERS)
	for i := 0; i < NUM_WORKERS; i++ {
		workers_consumption[i] = make([]int64, MAX_MOLD)
	}
}

func generate_random_nodes() {
	var biomeNames []string
	if TRANSLATE {
		biomeNames = []string{
			"Уголок интроверта", "Северный пшик", "Рай для плесени", "Мечта мотылька", 
			"Запретная лампа", "Ядерный сыр", "Умирающий пиксель", "Светящаяся лужа", 
			"Укол зонтиком", "Проспект слизи", "Студенческий холодильник", "Кладбище контейнеров", 
			"Мокрый носок", "Сгоревшая пицца", "Грязная клавиатура", "Великое ничто", 
			"Осколок логики", "Тихий омут", "Бескрайняя лень", "Процедурное болото", 
			"Драматический театр", "Центральная пробка", "Сосед с перфоратором", "Заблудший пиксель", 
			"Ошибка 404", "Точка возрождения", "Одинокий пельмень", "Просроченный йогурт", 
			"Прошлогодний салат", "Немытая кружка", "Забытый сыр", "Угол для наказаний", 
			"Зона потери WiFi", "Чашка Петри Альфа", "Нижний интернет", "Подкроватный монстр", 
			"Квантовая лужа", "Деление на ноль", "Кошачья лазерная указка", "Внезапный кипятильник",
			"Разогнанный процессор",
		}
	} else {
		biomeNames = []string{
			"Introvert Corner", "Northern Poof", "Mold Paradise", "Moth's Dream", 
			"The Forbidden Lamp", "Nuclear Cheese", "Dying Pixel", "Glowing Puddle", 
			"Needle Point", "Slime Avenue", "Student's Fridge", "Tupperware Grave", 
			"Wet Sock", "Burnt Pizza", "Dirty Keyboard", "The Great Nothing", 
			"Shard of Logic", "Still Waters", "Endless Laziness", "Procedural Swamp", 
			"Drama Theater", "Traffic Jam", "Noisy Neighbor", "Lost Pixel", 
			"Error 404", "Spawn Point", "Lonely Dumpling", "Expired Yogurt", 
			"Last Year's Salad", "Unwashed Mug", "Forgotten Cheese", "Timeout Corner", 
			"WiFi Drop Zone", "Petri Dish Alpha", "The Lower Internet", "Underbed Monster", 
			"Quantum Puddle", "Divided by Zero", "Cat's Laser Pointer", "Sudden Boiler",
			"Overclocked CPU",
		}		
	}
	
	nodes = make([]LightNode, 40)
	for i := 0; i < 40; i++ {
		nodes[i] = LightNode{
			Name:  biomeNames[i],
			X:     rand.Intn(SIZE_X),
			Y:     rand.Intn(SIZE_Y),
			Power: 5 + rand.Intn(12),
		}
	}
}

func generate_light_map() {
	light_map = make([]int8, SIZE_X*SIZE_Y)

	type FastNode struct {
		X, Y int
		MaxDistSq  float64 
		CoreDistSq float64 
		LogMax     float64 
	}
	
	fastNodes := make([]FastNode, len(nodes))
	for i, n := range nodes {
		maxDist := float64(n.Power * 50) 
		maxDistSq := maxDist * maxDist
		coreDist := float64(n.Power * 10) 
		coreDistSq := coreDist * coreDist

		fastNodes[i] = FastNode{
			X: n.X, 
			Y: n.Y, 
			MaxDistSq: maxDistSq,
			CoreDistSq: coreDistSq,
			LogMax: math.Log((maxDistSq - coreDistSq) + 1.0), 
		}
	}

	for x := 0; x < SIZE_X; x++ {
		for y := 0; y < SIZE_Y; y++ {
			maxLight := 0

			for _, node := range fastNodes {
				dx := abs(x - node.X)
				if dx > SIZE_X/2 { dx = SIZE_X - dx }

				dy := abs(y - node.Y)
				if dy > SIZE_Y/2 { dy = SIZE_Y - dy }

				distSq := float64(dx*dx + dy*dy)

				if distSq < node.MaxDistSq {
					decay := math.Log(distSq + 1.0) / node.LogMax
					
					nodeLight := 20 - int(20.0 * decay)

					if nodeLight > maxLight {
						maxLight = nodeLight
					}
				}
			}

			light_map[x*SIZE_Y+y] = int8(maxLight)
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
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


func save_world() {
	if is_saving {
		return 
	}
	is_saving = true

	if TRANSLATE {
		text_string = "Идёт фоновое сохранение мира..."
	} else {
		text_string = "Saving world in background..."
	}
	text_time = world_time
	
	was_paused := pause
	pause = true
	time.Sleep(50 * time.Millisecond)

	cellsCopy := make([]Cell, len(cells))
	copy(cellsCopy, cells)

	moldsCopy := make([]Mold, len(molds))
	copy(moldsCopy, molds[:]) 
	
	nodesCopy := make([]LightNode, len(nodes))
	copy(nodesCopy, nodes)

	wtCopy := int64(world_time)
	elCopy := int64(energy_light)

	pause = was_paused

	go func() {
		defer func() { is_saving = false }() 

		tmp_filename := "CuteMoldSave_tmp.gz"
		file, err := os.Create(tmp_filename)
		if err != nil {
			fmt.Println("Save error:", err)
			return
		}

		gz := gzip.NewWriter(file)

		nodesJSON, _ := json.Marshal(nodesCopy)
		lenNodes := int64(len(nodesJSON))

		binary.Write(gz, binary.LittleEndian, wtCopy)
		binary.Write(gz, binary.LittleEndian, elCopy)
		binary.Write(gz, binary.LittleEndian, lenNodes)
		
		gz.Write(nodesJSON)

		cellsBytes := unsafe.Slice((*byte)(unsafe.Pointer(&cellsCopy[0])), len(cellsCopy)*int(unsafe.Sizeof(cellsCopy[0])))
		gz.Write(cellsBytes)

		moldsBytes := unsafe.Slice((*byte)(unsafe.Pointer(&moldsCopy[0])), len(moldsCopy)*int(unsafe.Sizeof(moldsCopy[0])))
		gz.Write(moldsBytes)

		gz.Close()
		file.Close()

		os.Rename(tmp_filename, "CuteMoldSave.gz")

		if TRANSLATE {
			text_string = "Сохранение мира успешно выполнено!"
		} else {
			text_string = "Background save completed successfully!"
		}
		text_time = world_time
	}()
}

func load_world() {
	pause = true
	time.Sleep(10 * time.Millisecond)
	
	file, err := os.Open("CuteMoldSave.gz")
	if err != nil {
		if TRANSLATE {
			text_string = "Файл с сохранением не найден!"
		} else {
			text_string = "Save file not found!"
		}
		text_time = world_time
		pause = false
		return
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		if TRANSLATE {
			text_string = "Ошибка чтения из архива с сохранением!"
		} else {
			text_string = "Error reading save archive!"
		}
		text_time = world_time
		pause = false
		return
	}
	defer gz.Close()

	var wt, el, lenNodes int64
	binary.Read(gz, binary.LittleEndian, &wt)
	binary.Read(gz, binary.LittleEndian, &el)
	binary.Read(gz, binary.LittleEndian, &lenNodes)
	
	world_time = int(wt)
	energy_light = int(el)

	nodesJSON := make([]byte, lenNodes)
	io.ReadFull(gz, nodesJSON)
	json.Unmarshal(nodesJSON, &nodes)
	generate_light_map()

	cellsBytes := unsafe.Slice((*byte)(unsafe.Pointer(&cells[0])), len(cells)*int(unsafe.Sizeof(cells[0])))
	io.ReadFull(gz, cellsBytes)

	moldsBytes := unsafe.Slice((*byte)(unsafe.Pointer(&molds[0])), len(molds)*int(unsafe.Sizeof(molds[0])))
	io.ReadFull(gz, moldsBytes)
	
	if TRANSLATE {
		text_string = "Мир загружен из сохранения CuteMoldSave.gz!"
	} else {
		text_string = "World loaded from CuteMoldSave.gz!"
	}
	text_time = world_time

	show_starting_text = false 
	pause = false
}

func export_map_to_png() {
	if is_saving { return }
	is_saving = true
	
	if TRANSLATE {
		text_string = "Идёт экспорт карты в PNG..."
	} else {
		text_string = "Exporting map to PNG..."
	}
	text_time = world_time

	go func() {
		defer func() { is_saving = false }()
		
		img := image.NewRGBA(image.Rect(0, 0, SIZE_X, SIZE_Y))

		for x := 0; x < SIZE_X; x++ {
			for y := 0; y < SIZE_Y; y++ {
				idx := x*SIZE_Y + y
				
				pixIdx := (y*SIZE_X + x) * 4

				if cells[idx].mold != 0 {
					moldID := cells[idx].mold
					
					if cells[idx].spore && cells[idx].time > TIME_SPORE {
						img.Pix[pixIdx]   = 255
						img.Pix[pixIdx+1] = 255
						img.Pix[pixIdx+2] = 255
					} else {
						img.Pix[pixIdx]   = molds[moldID].rc
						img.Pix[pixIdx+1] = molds[moldID].gc
						img.Pix[pixIdx+2] = molds[moldID].bc
					}
				} else {
					local_light := min(int(energy_light), int(light_map[idx]))
					color_val := local_light * 8
					if color_val > 255 { color_val = 255 }
					bg_c := byte(color_val)
					
					img.Pix[pixIdx]   = bg_c
					img.Pix[pixIdx+1] = bg_c
					img.Pix[pixIdx+2] = bg_c
				}
				
				img.Pix[pixIdx+3] = 0xff
			}
		}

		filename := fmt.Sprintf("CuteMoldMap_%d.png", time.Now().Unix())
		file, err := os.Create(filename)
		if err != nil {
			fmt.Println("Save map error:", err)
			return
		}
		defer file.Close()

		png.Encode(file, img)

		if TRANSLATE {
			text_string = "Карта успешно сохранена в " + filename
		} else {
			text_string = "Map successfully saved to " + filename
		}
		text_time = world_time
	}()
}


func found_new_mold(x int) int {
	if x == -1 {
		new_mold := MAX_MOLD-1
		for new_mold > 1 {
			if molds[new_mold].free == true { 
				molds[new_mold].free = false
				return new_mold
			}
			new_mold--
		}
	} else {
		chunkSize := SIZE_X / NUM_WORKERS
		chunkX := x / chunkSize
		if chunkX >= NUM_WORKERS {
			chunkX = NUM_WORKERS - 1
		}
		
		new_mold := chunkX
		if new_mold == 0{
			new_mold = NUM_WORKERS
		}

		for new_mold < MAX_MOLD {
			if molds[new_mold].free == true {
				molds[new_mold].free = false
				return new_mold
			}
			new_mold += NUM_WORKERS
		}	
	}
	
	if TRANSLATE {
		text_string = "Паника! Слишком много плесеней!"
	} else {
		text_string = "Panic! Too much mold!"
	}
	text_time = world_time
	return 0
}

func generate_new_mold(x, y int) {
	idx := getCellIdx(x, y)

	if cells[idx].mold != 0 {
		return
	}

	local_light := min(energy_light, int(light_map[idx]))
	if local_light <= 8 {
		return
	}

	new_mold := found_new_mold(-1)
	
	molds[new_mold].name = generate_name()
	molds[new_mold].genome.RandomizeHalfZeros()
	molds[new_mold].energy = int64(ENERGY_DAY)
	molds[new_mold].leave = true
	molds[new_mold].free = false
	molds[new_mold].r, molds[new_mold].g, molds[new_mold].b = rand_color()
	molds[new_mold].rc = molds[new_mold].r
	molds[new_mold].gc = molds[new_mold].g
	molds[new_mold].bc = molds[new_mold].b
	
	cells[idx].mold = uint32(new_mold)
	cells[idx].time = 0
	cells[idx].n = 0
	cells[idx].dx = 1
	cells[idx].dy = 0
	t := molds[int(cells[idx].mold)].genome.Get(int(cells[idx].n)*4+3)
	cells[idx].spore = (SPORE_N <= t && t <= SPORE_K)
}

func load_genom(x, y int) {
	idx := getCellIdx(x, y)

	if cells[idx].mold != 0 {
		return
	}

	new_mold := found_new_mold(-1)
	
	molds[new_mold].name = generate_name()
	molds[new_mold].genome = mouse_gen
	molds[new_mold].energy = int64(ENERGY_DAY)
	molds[new_mold].leave = true
	molds[new_mold].free = false
	molds[new_mold].r, molds[new_mold].g, molds[new_mold].b = rand_color()
	molds[new_mold].rc = molds[new_mold].r
	molds[new_mold].gc = molds[new_mold].g
	molds[new_mold].bc = molds[new_mold].b
	
	cells[idx].mold = uint32(new_mold)
	cells[idx].time = 0 
	cells[idx].n = 0 
	cells[idx].dx = 1 
	cells[idx].dy = 0 
	t := molds[int(cells[idx].mold)].genome.Get(int(cells[idx].n)*4+3)
	cells[idx].spore = (SPORE_N <= t && t <= SPORE_K)
}

func create_new_mold(x, y int) {
	idx := getCellIdx(x, y)
	last_mold := cells[idx].mold
	new_mold := found_new_mold(x)
	
	molds[new_mold].genome = molds[last_mold].genome
	molds[new_mold].energy = int64(ENERGY_DAY)
	molds[new_mold].leave = true
	molds[new_mold].free = false
	
	if rand.Intn(MUTATE) == 0 {
		molds[new_mold].name = generate_name()
		molds[new_mold].genome.MutateOne()
		molds[new_mold].r, molds[new_mold].g, molds[new_mold].b = rand_color()
		molds[new_mold].rc = molds[new_mold].r
		molds[new_mold].gc = molds[new_mold].g
		molds[new_mold].bc = molds[new_mold].b
		
	} else {
		molds[new_mold].name = molds[last_mold].name
		molds[new_mold].r = molds[last_mold].r
		molds[new_mold].g = molds[last_mold].g 
		molds[new_mold].b = molds[last_mold].b
		rInt := int(molds[new_mold].r) + rand.Intn(20) - 10
		gInt := int(molds[new_mold].g) + rand.Intn(20) - 10
		bInt := int(molds[new_mold].b) + rand.Intn(20) - 10
		molds[new_mold].rc = byte(max(0, min(255, rInt)))
		molds[new_mold].gc = byte(max(0, min(255, gInt)))
		molds[new_mold].bc = byte(max(0, min(255, bInt)))	
	}
	
	cells[idx].mold = uint32(new_mold)
	cells[idx].time = 0
	cells[idx].n = 0
	t := molds[int(cells[idx].mold)].genome.Get(int(cells[idx].n)*4+3)
	cells[idx].spore = (SPORE_N <= t && t <= SPORE_K)
}

func add_cell(x, y, x2, y2, n int) {
	idx := getCellIdx(x2,y2)
	
	if cells[idx].mold == 0 { 
		cells[idx].mold = cells[getCellIdx(x,y)].mold
		cells[idx].n = int16(n)
		cells[idx].time = 0
		cells[idx].dx = int8(mod_x(x2-x+1) - 1)
		cells[idx].dy = int8(mod_y(y2-y+1) - 1)
		t := molds[int(cells[idx].mold)].genome.Get(int(cells[idx].n)*4+3)
		cells[idx].spore = (SPORE_N <= t && t <= SPORE_K)
	}
}

func neighbor(x, y, dx, dy, dir int) (int, int) {
	if (dx == 1 && dir == 0) || (dx == -1 && dir == 2) || (dy == 1 && dir == 1) {
		return x, mod_y(y + 1)
	}
	if (dx == 1 && dir == 1) || (dy == 1 && dir == 2) || (dy == -1 && dir == 0) {
		return mod_x(x + 1), y
	}
	if (dx == 1 && dir == 2) || (dx == -1 && dir == 0) || (dy == -1 && dir == 1) {
		return x, mod_y(y - 1)
	}
	if (dx == -1 && dir == 1) || (dy == 1 && dir == 0) || (dy == -1 && dir == 2) {
		return mod_x(x - 1), y
	}
	if TRANSLATE {
		text_string = "Паника! Не могу найти моего соседа!"
	} else {
		text_string = "Panic! I can't find my neighbor!"
	}
	text_time = world_time
	return 0, 0
}

func growth_cell(x, y int) {
	idx := getCellIdx(x, y) 
	n := cells[idx].n       
	
	for dir := 0; dir < 3; dir++ {
		val := molds[int(cells[idx].mold)].genome.Get(int(n)*4 + dir)
		
		if val <= LEN_GROWTH {
			next_n := (int(n) + int(val) - 27 + LEN_GENOME) & (LEN_GENOME - 1)
			
			x2, y2 := neighbor(x, y, int(cells[idx].dx), int(cells[idx].dy), dir)
			add_cell(x, y, x2, y2, int(next_n))
		}
	}
}


func delete_all() {
	pause = true
	time.Sleep(50 * time.Millisecond)
	
	for index := 0; index < MAX_MOLD; index++ {
		molds[index].leave = false
	}
	
	for i := range cells {
		cells[i].mold = 0
	}
	
	generate_random_nodes()
	generate_light_map()
	
	update_lightmap_image()
	
	calc_nodes_text_width()
	
	os.Remove("CuteMoldSave.gz")
	
	if TRANSLATE {
		text_string = "Сохранение удалено. Новый мир сгенерирован!"
	} else {
		text_string = "Save deleted. New world generated!"
	}
	text_time = world_time

	pause = false
}

func update() {
	if !pause {
		if atomic.CompareAndSwapInt32(&isSweeping, 0, 1) {
			go sweep_cavities_background()
		}
	
		var wg sync.WaitGroup
		chunkSize := SIZE_X / NUM_WORKERS

		for w := 0; w < NUM_WORKERS; w++ {
			for i := 0; i < MAX_MOLD; i++ {
				workers_consumption[w][i] = 0
			}
		}

		for w := 0; w < NUM_WORKERS; w++ {
			startX := w * chunkSize
			endX := (w + 1) * chunkSize
			if w == NUM_WORKERS-1 { endX = SIZE_X }

			wg.Add(1)
			go func(s, e, worker_id int) {
				defer wg.Done()
				for x := s; x < e; x++ {
					idx := x * SIZE_Y
					
					_ = cells[idx + SIZE_Y - 1] 

					for y := 0; y < SIZE_Y; y++ {
						if cells[idx].mold != 0 {
							consumption := int64(ENERGY_DAY)
							if cells[idx].spore {
								consumption += int64(ENERGY_SPORE)
							}
							consumption = consumption * int64(cells[idx].time/TIME_CELL)
							
							workers_consumption[worker_id][cells[idx].mold] += consumption						
							
							cells[idx].time++
						}
						idx++
					}
				}
			}(startX, endX, w)
		}
		wg.Wait()

		for index := 0; index < MAX_MOLD; index++ {
			if !molds[index].leave {
				molds[index].free = true
			} else {
				var total_loss int64 = 0
				for w := 0; w < NUM_WORKERS; w++ {
					total_loss += workers_consumption[w][index]
				}
				
				molds[index].energy -= total_loss
				molds[index].energy += molds[index].cavity_energy

				if molds[index].energy < 0 {
					molds[index].leave = false
				}
			}
		}

		for w := 0; w < NUM_WORKERS; w++ {
			startX := w * chunkSize
			endX := (w + 1) * chunkSize
			if w == NUM_WORKERS-1 {
				endX = SIZE_X
			}

			wg.Add(1)
			go func(s, e int) {
				defer wg.Done()
				for x := s; x < e; x++ {
					idx := x * SIZE_Y

					_ = cells[idx + SIZE_Y - 1]

					for y := 0; y < SIZE_Y; y++ {
						if cells[idx].mold != 0 {
							if molds[int(cells[idx].mold)].leave {	
								if cells[idx].time > 0 {
									growth_cell(x, y)
								}
							} else {
								if cells[idx].spore && cells[idx].time >= TIME_SPORE{ 
									create_new_mold(x, y)
								} else {
									cells[idx].mold = 0
								}
							}
						}
						idx++
					}
				}
			}(startX, endX)
		}
		wg.Wait()		
	}
}

func key_press() {
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		for i := 0; i < 10000; i++ {		
			generate_new_mold(rand.Intn(SIZE_X), rand.Intn(SIZE_Y))
		}
		show_starting_text = false
	}
	
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		pause = !pause
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		save_world()
	}
	
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
	 	load_world()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		delete_all()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		show_inform_control = !show_inform_control
	}
	
	if inpututil.IsKeyJustPressed(ebiten.KeyJ) {
		show_inform_technical = !show_inform_technical
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyH) && DEVELOPER {
		show_world = !show_world
	}
	
	if inpututil.IsKeyJustPressed(ebiten.KeyM) && DEVELOPER {
		show_light_map = !show_light_map
	}
	
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		export_map_to_png()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF) && DEVELOPER {
		fmt.Println("Данные")
		fmt.Println("FPS: ", ebiten.CurrentFPS())
	}
}

func mouse_click() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if 0 <= x && x < WIN_X && 0 <= y && y < WIN_Y {
			x = mod_x(zoom_x + int(x/zoom))
			y = mod_y(zoom_y + int(y/zoom))
			
			
			if cells[getCellIdx(x,y)].mold != 0 {
				compressed := compress_genome(&molds[int(cells[getCellIdx(x,y)].mold)].genome)
				clipboard.WriteAll(compressed)
				
				if TRANSLATE {
					text_string = "Геном '" + string(molds[int(cells[getCellIdx(x,y)].mold)].name[:]) + "' скопирован."
				} else {
					text_string = "Genome '" + string(molds[int(cells[getCellIdx(x,y)].mold)].name[:]) + "' copied."
				}
				text_time = world_time				
			} else {
				save, _ := clipboard.ReadAll()
				
				err := decompress_genome(save, &mouse_gen)
				if err == nil {
					load_genom(x, y)
					if TRANSLATE {
						text_string = "Геном успешно загружен и посажен!"
					} else {
						text_string = "Genome loaded and planted!"
					}
					text_time = world_time	
					show_starting_text = false
				} else {
					if TRANSLATE {
						text_string = "Ошибка: неверный формат генома."
					} else {
						text_string = "Error: invalid genome format."
					}
					text_time = world_time
				}
			}
			
		} else {
			if TRANSLATE {
				text_string = "Клик вне мира."
			} else {
				text_string = "Click out of the world."
			}
			text_time = world_time
		}
	}

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
	
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle) {
		x, y := ebiten.CursorPosition()
		if 0 <= x && x < WIN_X && 0 <= y && y < WIN_Y {
			center_x := mod_x(zoom_x + int(x/zoom))
			center_y := mod_y(zoom_y + int(y/zoom))
			
			radius := 50 
			rSq := radius * radius

			for dx := -radius; dx <= radius; dx++ {
				for dy := -radius; dy <= radius; dy++ {
					if dx*dx+dy*dy <= rSq {
						nx := mod_x(center_x + dx)
						ny := mod_y(center_y + dy)
						
						idx := getCellIdx(nx, ny)
						
						molds[cells[idx].mold].leave = false		

						cells[idx].spore = false
						cells[idx].mold = 0
					}
				}
			}

			if TRANSLATE {
				text_string = "Плесень в радиусе поражения уничтожена!"
			} else {
				text_string = "Mold in the specified radius destroyed!"
			}
			text_time = world_time
		}
	}
}

func (g *Game) Update() error {
	if ebiten.IsWindowMinimized() {
		return nil
	}
	
	mouse_click()
	key_press()
	
	if atomic.CompareAndSwapInt32(&isUpdating, 0, 1) {
		go func() {
			update()
			world_time++
			atomic.StoreInt32(&isUpdating, 0)
		}()
	}
	return nil
}

func (g *Game) getLightAtMouse() int {	
	mx, my := ebiten.CursorPosition()

	worldX := mx / zoom
	worldY := my / zoom

	if mx < 0 || mx >= WIN_X || my < 0 || my >= WIN_Y {
		return 0
	}
	finalX := mod_x(worldX + zoom_x)
	finalY := mod_y(worldY + zoom_y)

	idx := finalX*SIZE_Y + finalY
	
	return min(energy_light, int(light_map[idx]))
}

func (g *Game) Draw(screen *ebiten.Image) {
	if ebiten.IsWindowMinimized() {
		return
	}

	if show_world {
		if g.pixels == nil {
			g.pixels = make([]byte, WIN_X*WIN_Y*4)
		}
		
		if g.moldsScreen == nil {
			g.moldsScreen = ebiten.NewImage(WIN_X, WIN_Y)
		}

		clear(g.pixels)

		limitX := int(WIN_X / zoom)
		limitY := int(WIN_Y / zoom)

		var wg sync.WaitGroup
		chunkSize := limitX / NUM_WORKERS 
		if chunkSize == 0 {
			chunkSize = 1
		}

		for w := 0; w < NUM_WORKERS; w++ {
			wg.Add(1)
			
			startX := w * chunkSize
			endX := (w + 1) * chunkSize
			
			if w == NUM_WORKERS-1 {
				endX = limitX
			}

			go func(startX, endX int) {
				defer wg.Done()
				
				for x := startX; x < endX; x++ {
					baseX := x * zoom
					x0 := mod_x(x + zoom_x)
					
					_ = cells[x0*SIZE_Y + SIZE_Y - 1]
					
					y0 := mod_y(zoom_y)
					
					for y := 0; y < limitY; y++ {
						baseY := y * zoom
						idx := x0*SIZE_Y + y0
						
						if cells[idx].mold != 0 {
							moldID := cells[idx].mold
							
							color_r := molds[moldID].rc
							color_g := molds[moldID].gc
							color_b := molds[moldID].bc
							
							isSpore := cells[idx].spore
							var spore_color byte = 0
							if isSpore && cells[idx].time > TIME_SPORE {
								spore_color = 255
							}
							
							for j := 0; j < zoom; j++ {
								pic := ((baseY + j)*WIN_X + baseX) * 4
								
								if isSpore {
									for i := 0; i < zoom; i++ {
										if j >= 2 && j < zoom-1 && i >= 2 && i < zoom-1 {
											g.pixels[pic]   = spore_color
											g.pixels[pic+1] = spore_color
											g.pixels[pic+2] = spore_color
										} else {
											g.pixels[pic]   = color_r
											g.pixels[pic+1] = color_g
											g.pixels[pic+2] = color_b
										}
										g.pixels[pic+3] = 0xff
										pic += 4 
									}
								} else {
									for i := 0; i < zoom; i++ {
										g.pixels[pic]   = color_r
										g.pixels[pic+1] = color_g
										g.pixels[pic+2] = color_b
										g.pixels[pic+3] = 0xff
										pic += 4 
									}
								}
							}
						}
						y0++
						if y0 >= SIZE_Y {
							y0 = 0
						}
					}
				}
			}(startX, endX)
		}
		
		wg.Wait()


		g.moldsScreen.ReplacePixels(g.pixels)

		if show_light_map {
			for i := 0; i < 2; i++ {
				for j := 0; j < 2; j++ {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(float64(-zoom_x + i*SIZE_X), float64(-zoom_y + j*SIZE_Y))
					op.GeoM.Scale(float64(zoom), float64(zoom))
					screen.DrawImage(lightMapImage, op)
				}
			}
		} else {
			screen.Fill(color.Black)
		}

		screen.DrawImage(g.moldsScreen, &ebiten.DrawImageOptions{})
		

		if show_light_map {
			biomeColor := color.RGBA{80, 200, 150, 255}

			for _, b := range nodes {
				dx := (b.X - zoom_x) & MASK_X
				dy := (b.Y - zoom_y) & MASK_Y

				if dx > SIZE_X/2 {
					dx -= SIZE_X
				}
				if dy > SIZE_Y/2 {
					dy -= SIZE_Y
				}

				screenX := dx * zoom
				screenY := dy * zoom

				if screenX > -200 && screenX < WIN_X+200 && screenY > -50 && screenY < WIN_Y+50 {
					centeredX := screenX - (b.TextWidth / 2)
					drawTextWithOutline(screen, b.Name, NormalFont, centeredX, screenY, biomeColor)
				}
			}
		}

		if show_starting_text {
			var infoText []string
			if !TRANSLATE {
				infoText = []string{
				"The mold gets energy from the empty cells it has surrounded.",
				"The longer the mold lives, the more energy it consumes.",
				"When its energy runs out, the mold dies.",
				"New molds emerge from its spores with the same genome and a possible point mutation.",
				"Spores require time to mature: mature spores are white, while immature ones are black.",
				" ",
				"Press G to generate new molds.",
				}
			} else {
				infoText = []string{
				"Плесень получает энергию от пустых клеток, которые она окружила.",
				"Чем дольше живёт плесень, тем больше энергии она потребляет.",
				"Когда энергия заканчивается, плесень умирает.",
				"Из её спор появляются новые плесени с таким же геномом и возможной точечной мутацией.",
				"Споры требуют время на созревание: созервшие споры белые, несозревшие - черные.",
				" ",
				"Нажми G чтобы сгенерировать новые плесени.",
				}
			}

			lineHeight := 30

			startY := int(WIN_Y/2) - (len(infoText)*lineHeight)/2 - 40

			for i, line := range infoText {
				bounds := text.BoundString(NormalFont, line)
				textWidth := bounds.Dx()
				
				x := (WIN_X - textWidth) / 2
				
				y := startY + (i * lineHeight)
				
				text.Draw(screen, line, NormalFont, x, y, color.White)
			}
		}

		if pause {
			drawTextWithOutline(screen, "PAUSE", NormalFont, WIN_X/2-20, WIN_Y/2, color.White)
		}

		if world_time-text_time < TIME_SHOW_NOTICE || is_saving {
			text.Draw(screen, text_string, NormalFont, 20, WIN_Y-20, color.White)
		}

		if show_inform_control && controlMenuImage != nil {
			op := &ebiten.DrawImageOptions{}
			screen.DrawImage(controlMenuImage, op)
		}		
		
		if show_inform_technical {	
			mx, my := ebiten.CursorPosition()
			currentLight := max(g.getLightAtMouse(),0)
			text.Draw(screen, fmt.Sprint(currentLight), NormalFont, mx+15, my+15, color.White)
		}
	} else {
		if TRANSLATE {
			text.Draw(screen, "Нажми H чтобы показать мир.", NormalFont, WIN_X/2-150, WIN_Y/2, color.White)
		} else {
			text.Draw(screen, "Press H to show world.", NormalFont, WIN_X/2-130, WIN_Y/2, color.White)
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	if outsideWidth <= 0 || outsideHeight <= 0 {
		return 1, 1
	}
	return WIN_X, WIN_Y
}

func main() {
	rand.Seed(time.Now().UnixNano())

	ebiten.SetWindowSize(WIN_X, WIN_Y)
	ebiten.SetWindowTitle("Cute Mold")

	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(logoData))
	logoImage, _, err := image.Decode(reader)
	if err != nil {
		panic(err)
	}
	ebiten.SetWindowIcon([]image.Image{logoImage})

	tt, _ := opentype.Parse(fonts.MPlus1pRegular_ttf)
	NormalFont, _ = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    20,
		DPI:     72,
		Hinting: font.HintingFull,
	})

	create_control_menu_image()

	if _, err := os.Stat("CuteMoldSave.gz"); err == nil {
		load_world()
		update_lightmap_image()
	} else {
		update_lightmap_image()
	}
	
	calc_nodes_text_width()

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
