io.output():setvbuf("no")
math.randomseed(os.time())
love.window.setTitle("cute mold")

lume = require "lume"

SIZE_X = 350
SIZE_Y = 200
ZOOM = 4

LEN_GENOM = 50
MUTATE = 30
NONE = 0
SPORE = -1

VISUAL = true
PAUSE = false

ENERGY_LIGHT = 3
ENERGY_DAY = 1
ENERGY_MOLD = 200

TIME_CELL = 100
TIME = 0

--        top    right   bottom   left
ROTATE = {{0,1}, {1,0}, {0,-1}, {-1,0}}

love.window.setMode(SIZE_X*ZOOM, SIZE_Y*ZOOM)
love.window.setVSync(1)

function randon_flag()
  return (math.random(2) == 1)
end

function chouse_gen()
  if math.random(LEN_GENOM) == 1 then
    return SPORE
  end
  if randon_flag() then
    return math.random(LEN_GENOM)
  end
  return NONE
end

function generate_mold(x,y,flag)
  if cells[x][y] == nil then
    local genom = #genoms + 1
    
    genoms[genom] = {}
    genoms[genom]["num"] = 1
    if flag then
      genoms[genom]["color"] = mouse_genom.color 
    else
      genoms[genom]["color"] = {math.random()/2+0.2, math.random()/2+0.2, math.random()/2+0.2}
    end

    for i=1, LEN_GENOM do
      genoms[genom][i] = {}
      for j=1, 4 do
        if flag then
          genoms[genom][i][j] = mouse_genom[i][j]
        else
          genoms[genom][i][j] = chouse_gen()
        end
      end
    end

    local mold = #molds + 1
    molds[mold] = {}
    
    molds[mold]["color"] = genoms[genom].color
    molds[mold]["gen"] = genom
     
    molds[mold]["e"] =  ENERGY_DAY*5
    molds[mold]["num"] = 1
    
    cells[x][y] = {}
    
    cells[x][y]["dx"] = 0
    cells[x][y]["dy"] = 1
    
    cells[x][y]["mold"] = mold
    cells[x][y]["n"] = 1
    
    cells[x][y]["sleep"] = true
    cells[x][y]["time"] = 0
  end
end

function new_molds(x,y)
  local genom1 = molds[cells[x][y].mold].gen

  local mold1 = cells[x][y].mold
  local mold2 = #molds + 1
  
  local dx = cells[x][y].dx
  local dy = cells[x][y].dy
  
  molds[mold2] = {}
  
  if math.random(MUTATE) == 1 then
    local genom2 = #genoms + 1
    
    genoms[genom2] = {}
    genoms[genom2]["num"] = 1
    genoms[genom2]["color"] = {math.random()/2+0.2, math.random()/2+0.2, math.random()/2+0.2}
    
    for i=1, LEN_GENOM do
      genoms[genom2][i] = {}
      for j=1, 4 do
        genoms[genom2][i][j] = genoms[genom1][i][j]
      end
    end
    for j=1, 4 do
      genoms[genom2][math.random(LEN_GENOM)][j] = chouse_gen()
    end
    
    molds[mold2]["gen"] = genom2
    molds[mold2]["color"] = genoms[genom2].color
    
  else
    genoms[genom1].num = genoms[genom1].num + 1
    
    molds[mold2]["gen"] = genom1
    local c = (math.random() - 0.5) / 4
    molds[mold2]["color"] = {genoms[genom1].color[1] + c, genoms[genom1].color[2] + c, genoms[genom1].color[3] + c}
  end
  
  molds[mold2]["e"] = cells[x][y].time
  molds[mold2]["num"] = 1
  
  cells[x][y] = {}
  
  cells[x][y]["dx"] = dx
  cells[x][y]["dy"] = dy
  
  cells[x][y]["mold"] = mold2
  cells[x][y]["n"] = 1
  
  cells[x][y]["sleep"] = true
  cells[x][y]["time"] = 0
end

function add_cell(x,y,x2,y2,n)
  if cells[x2][y2] == nil then
    if n == SPORE then
      if molds[cells[x][y].mold].e >= ENERGY_MOLD then
        molds[cells[x][y].mold].e = molds[cells[x][y].mold].e - ENERGY_MOLD
      else
        return 0
      end
    end
    
    molds[cells[x][y].mold].num = molds[cells[x][y].mold].num + 1
    
    cells[x2][y2] = {}
    
    cells[x2][y2]["dx"] = (x2-x+1)%SIZE_X - 1
    cells[x2][y2]["dy"] = (y2-y+1)%SIZE_Y - 1
    
    cells[x2][y2]["mold"] = cells[x][y].mold
    cells[x2][y2]["n"] = n
    
    cells[x2][y2]["sleep"] = true
    cells[x2][y2]["time"] = 0
  end
end


function neitherhood(x,y,dx,dy,dir)
  if ((dx==1 and dir==4) or (dx==-1 and dir==2) or (dy==1 and dir==1) or (dy==-1 and dir==3)) then
    return ((x-1)%SIZE_X+1), (y%SIZE_Y+1)
  
  elseif (dx==1 and dir==1) or (dx==-1 and dir==3) or (dy==1 and dir==2) or (dy==-1 and dir==4) then
    return ((x)%SIZE_X+1), ((y-1)%SIZE_Y+1)
    
  elseif (dx==1 and dir==2) or (dx==-1 and dir==4) or (dy==1 and dir==3) or (dy==-1 and dir==1) then
    return ((x-1)%SIZE_X+1), ((y-2)%SIZE_Y+1)
    
  elseif (dx==1 and dir==3) or (dx==-1 and dir==1) or (dy==1 and dir==4) or (dy==-1 and dir==2) then
    return ((x-2)%SIZE_X+1), ((y-1)%SIZE_Y+1)
  end
end  

function add_cells(x,y)
  if cells[x][y].n ~= SPORE then
    for i=1,4 do
      if genoms[molds[cells[x][y].mold].gen][cells[x][y].n][i] ~= NONE then
        x2,y2 = neitherhood(x, y, cells[x][y].dx, cells[x][y].dy, i)
        if not(BARIER) or (x2 ~= BARIER_X and x2 ~= 1 and y2 ~= BARIER_Y and y2 ~= 1) then
          add_cell(x,y,x2,y2, genoms[molds[cells[x][y].mold].gen][cells[x][y].n][i])
        end
      end
    end
  end
end

function photosynthesis(x,y)
  function ph(x2,y2)
    if cells[x2][y2] == nil then
      return ENERGY_LIGHT
    else
      return 0
    end
  end
  molds[cells[x][y].mold].e = molds[cells[x][y].mold].e + ph(x%SIZE_X+1,y) + ph((x-2)%SIZE_X+1,y) + ph(x,y%SIZE_Y+1) + ph(x,(y-2)%SIZE_X+1)
end

function delete_cell(x,y)
  local mold = cells[x][y].mold 
  if cells[x][y].n == SPORE then
    new_molds(x,y)
  else
    cells[x][y] = nil
  end
  molds[mold].num = molds[mold].num - 1
  if molds[mold].num < 1 then
    local genom = molds[mold].gen
    molds[mold] = nil
    genoms[genom].num = genoms[genom].num - 1
    if genoms[genom].num < 1 then
      genoms[genom] = nil
    end
  end
end

function information()
  print("time", TIME)
  local num_g = 0
  local num_m = 0
  for i=1, 1000 do
    if genoms[i] ~= nil then
      num_g = num_g + 1
    end
  end
  for i=1, 5000 do
    if molds[i] ~= nil then
      num_m = num_m + 1
    end
  end
  print("genomes", num_g, "molds", num_m)
end

function love.load()
  cells = {}
  for i=1, SIZE_X do
    for j=1, SIZE_Y do
      cells[i] = {}
      cells[i][j] = nil    
    end
  end

  molds = {}
  genoms = {}

  mouse_genom = {}
  m_i0 = 0
  m_j0 = 0
end

function love.update()  
  mouse_click()
  if not PAUSE then
    TIME = TIME+1
    
    for x=1,SIZE_X do
      for y=1, SIZE_Y do
        if cells[x][y] then
          cells[x][y].sleep = false
          cells[x][y].time = cells[x][y].time + 1
          molds[cells[x][y].mold].e = molds[cells[x][y].mold].e - ENERGY_DAY * (1 + cells[x][y].time/TIME_CELL)
          if cells[x][y].n ~= SPORE then
            photosynthesis(x,y)
          end
        end
      end
    end
    
    for x=1, SIZE_X do
      for y=1, SIZE_Y do
        if cells[x][y] then
          if molds[cells[x][y].mold].e < 0 then
            delete_cell(x,y)
          end
        end
      end
    end

    for x=1,SIZE_X do
      for y=1, SIZE_Y do
        if cells[x][y] then
          if cells[x][y].sleep == false then
            add_cells(x,y)
          end
        end
      end
    end
  end
end 

function love.keypressed(key)
  if key == 'v' then
    VISUAL = not(VISUAL)
  
  elseif key == 'p' then
    PAUSE = not(PAUSE)
  
  elseif key == "g" then
    for i=1,300 do
      generate_mold(math.random(SIZE_X),math.random(SIZE_Y), false)
    end
    
  elseif key == "q" then
    ENERGY_LIGHT = ENERGY_LIGHT - 0.1
    print("light", ENERGY_LIGHT)
  elseif key == "w" then
    ENERGY_LIGHT = ENERGY_LIGHT + 0.1
    print("light", ENERGY_LIGHT)

  elseif key == 'f' then
    print(love.timer.getFPS())
  elseif key == "i" then
    information()
    
  elseif key == "escape" then
      love.window.close()
      love.event.quit()
  end
end

function mouse_click()
  if love.mouse.isDown(1) then
    local i0 = math.floor(love.mouse.getX()/ZOOM) + 1
    local j0 = math.floor(love.mouse.getY()/ZOOM) + 1
    
    if i0 ~= m_i0 or j0 ~= m_j0 then
      m_i0 = i0
      m_j0 = j0
      
      if cells[i0][j0] ~= nil then
        love.system.setClipboardText(lume.serialize(genoms[molds[cells[i0][j0].mold].gen]))
        print("genome saved to the clipboard")
      
      else
        local a = love.system.getClipboardText()
        if string.byte(a, 1) == 123 then
          mouse_genom = lume.deserialize(a)
          generate_mold(i0,j0,true)
        end
      end
    end
  end
end

function love.draw()  
  if VISUAL then
    for i=1, SIZE_X do
      for j=1, SIZE_Y do
        if cells[i][j] ~= nil then
          if molds[cells[i][j].mold] ~= nil then
            love.graphics.setColor((molds[cells[i][j].mold].color))
            love.graphics.rectangle("fill", i*ZOOM-ZOOM, j*ZOOM-ZOOM, ZOOM, ZOOM)
            if cells[i][j].n == SPORE then
              love.graphics.setColor(0,0,0)
              love.graphics.rectangle("fill", i*ZOOM-ZOOM/4*3, j*ZOOM-ZOOM/4*3, ZOOM/2, ZOOM/2)
            end
          end
        end
      end
    end
  end
end
