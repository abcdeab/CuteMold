# Сute mold
Mold grow and evolution simulation. The mold gets energy for the void space around it, so the mold that has more void will survive. Molds cannot interact or damage each other, evolutionary selection goes only by competing for space.

Molds with the same genome have the same color, but different shades. Mold with a mutated genome get a new color. Cells with black dots are spores. 

G to generate new molds. D to delete all the molds. Q/W to decrease/increase energy of the voids. P to pause. F to fullscreen. Mouse wheel to zoom. Right mouse button to move the screen. Left mouse click to copy a genome to clipboard. Click on the empty space to create a new mold with the genome saved in your clipboard.

You can share your molds in the comments by pasting their genomes as text! Show off your best molds! To see other people's molds, copy their genome to the clipboard, run the simulation, and click on a void. Give your molds time to become beautiful. Evolution isn't a quick process!

![image](https://user-images.githubusercontent.com/108512083/177539565-39ab3136-3d84-47aa-900e-3da9efcd708f.png)
![pic1](https://user-images.githubusercontent.com/108512083/177720197-5578ffeb-b221-4fbf-a52a-8313ba533f46.png)

---

Requires [ebiten](https://github.com/rxi/lume) and [clipboard](github.com/atotto/clipboard). Thanks for making them!
```
go get github.com/atotto/clipboard
go get github.com/hajimehoshi/ebiten/v2
```
