package main

import "fmt"
import "os"
import "os/exec"
import "io/ioutil"
import "io"
import "strings"
import "golang.org/x/image/draw"
import "image/jpeg"
import "image/png"
import "image"

var inkscapeBins = []string{"/Applications/Inkscape.app/Contents/MacOS/inkscape",
    "/usr/bin/inkscape",
    "/snap/bin/inkscape",
   }
var inkscapeBin = ""

var force = false
func isOlder(src string, than string) bool {
	if force {
		return false
	} 
	stat, err := os.Stat(src)

	// If the file does not exist, it is not older.
	if os.IsNotExist(err) {
		return false
	}

	statThan, _ := os.Stat(than)

	return stat.ModTime().After(statThan.ModTime())
}

func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func rescale(ssrc string, sdst string) {
	
	input, _ := os.Open(ssrc)
defer input.Close()

output, err := os.Create(sdst)
defer output.Close()
if err != nil {
	panic(fmt.Sprintf("Error creating %s", sdst))
		
}

// Decode the image (from PNG to image.Image):
var src image.Image
if strings.Contains(ssrc, ".png") {
   src, err = png.Decode(input) 
} else {
	src, err = jpeg.Decode(input) 
}

if err != nil {
	panic(fmt.Sprintf("Error loading %s", ssrc))
		
}

// Set the expected size that you want:
dst := image.NewRGBA(image.Rect(0, 0, 100, 100.0 * src.Bounds().Max.Y / src.Bounds().Max.X))

// Resize:
draw.NearestNeighbor.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)

// Encode to `output`: 
png.Encode(output, dst)
}

func prepareImg(src string, dst string) {
	// Compare last modified and skip if necessary
	if isOlder(dst, src) {
		return
	}

	if mode == "release" {
		// Copy
		fmt.Println("Copy:", src, "->", dst)
		copy(src, dst)
	} else {
		fmt.Println("Scale:", src, "->", dst)
		rescale(src, dst)
	}
}

func prepareDrawing(src string, dst string) {
	dst = strings.ReplaceAll(dst, ".svg", ".png")

	// Compare last modified and skip if necessary
	if isOlder(dst, src) {
		return
	}

	bin := inkscapeBin
	arg0 := "--export-filename=" + dst
	var arg1 = "--export-dpi=300"
	if mode == "debug" {
		arg1 = "--export-dpi=100"
	}
	arg2 := src

	fmt.Println(inkscapeBin, arg0, arg1, arg2)

	out, err := exec.Command(bin, arg0, arg1, arg2).CombinedOutput()

	if err != nil {
		fmt.Println("%s %s", string(out), err)
		return
	}
}

func prepare(folder string, f func(string, string)) {
	os.MkdirAll(base+"/"+folder, os.ModePerm)
	items, _ := ioutil.ReadDir(folder)
	for _, item := range items {
		if item.IsDir() {
			fmt.Println("Subdirectores in '%s' are not supported", folder)
			os.Exit(1)
		}

		var src = folder + item.Name()
		var dst = base + "/" + folder + item.Name()
		f(src, dst)
	}
}

func findInkscape() {
   for _, candidate := range inkscapeBins {
   	if _, err := os.Stat(candidate); err != nil {
        continue
    }
    inkscapeBin = candidate
   	break
   	}
}

func makeCover(src string, dst string) {
	if isOlder(dst, src) {
		return
	}
	bin := inkscapeBin
	arg0 := "--export-filename=" + dst
	var arg1 = "--export-dpi=300"
	if mode == "debug" {
		arg1 = "--export-dpi=100"
	}
	arg2 := "--export-type=pdf"
	arg3 := "--export-text-to-path"
	arg4 := src

	fmt.Println(inkscapeBin, arg0, arg1, arg2, arg3, arg4)

	out, err := exec.Command(bin, arg0, arg1, arg2, arg3, arg4).CombinedOutput()
	if err != nil {
		fmt.Println("%s %s", string(out), err)
		return
	}
}

var mode = "debug"
var base = ""

func main() {
   findInkscape()

   fmt.Println("Building...")
	var args = os.Args

	if len(args) > 1 {
		mode = args[1]
	}

	if len(args) > 2 {
		force = true
	}

	if mode != "debug" && mode != "release" {
		fmt.Println("Mode must be either 'debug' or 'release'.")
		return
	}

	base = "out/" + mode
	os.MkdirAll(base, os.ModePerm)
    
    makeCover("cover/cover.svg", base + "/illu/cover.pdf")

	prepare("illu/img/", prepareImg)
	prepare("illu/d/", prepareDrawing)

	bin := "pdflatex"
	arg0 := "-output-directory"
	arg1 := "out"
	arg2 := `\def\base{` + base + `} \input{src/book.tex}`

	out, err := exec.Command(bin, arg0, arg1, arg2).CombinedOutput()

	if err != nil {
		fmt.Println("%s %s", string(out), err)
		return
	}
}