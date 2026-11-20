// addnoise adds noise to an image the way Stable Diffusion 3 does it
// during training (rectified flow):
//
//	x_t = (1 - t) * x0 + t * eps
//
// x0  = clean image, scaled to [-1, 1]
// eps = Gaussian noise, mean 0, standard deviation 1
// t   = time in [0, 1]. SD3 uses 1000 steps, so step 500 means t = 0.5.
//
// The same eps is used for every step, so the ten outputs form one smooth
// sequence: the picture fades into the same noise pattern.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"
)

const totalSteps = 1000 // SD3 uses 1000 timesteps

// buildSteps returns n timesteps spread evenly over 1..totalSteps.
// n=10 -> 100, 200, ..., 1000.  n=20 -> 50, 100, ..., 1000.
// Step 0 (the clean image) is not included.
func buildSteps(n int) []int {
	steps := make([]int, n)
	for i := 1; i <= n; i++ {
		steps[i-1] = (i*totalSteps + n/2) / n // rounded
	}
	return steps
}

func main() {
	in := flag.String("in", "sample.png", "input image")
	outDir := flag.String("out", "noisy", "output directory")
	seed := flag.Uint64("seed", 42, "random seed (same seed = same noise)")
	n := flag.Int("steps", 10, fmt.Sprintf("number of frames, 1..%d", totalSteps))
	flag.Parse()

	if *n < 1 || *n > totalSteps {
		log.Fatalf("-steps must be between 1 and %d, got %d", totalSteps, *n)
	}
	steps := buildSteps(*n)

	src, err := load(*in)
	if err != nil {
		log.Fatal(err)
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	// x0: clean image in [-1, 1], 3 channels (R, G, B) per pixel.
	x0 := make([]float32, w*h*3)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(b.Min.X+x, b.Min.Y+y)
			o := (y*w + x) * 3
			for c := 0; c < 3; c++ {
				x0[o+c] = float32(src.Pix[i+c])/255*2 - 1
			}
		}
	}

	// eps: one fixed Gaussian noise field, reused for every step.
	rng := rand.New(rand.NewPCG(*seed, *seed))
	eps := make([]float32, len(x0))
	for i := range eps {
		eps[i] = float32(rng.NormFloat64())
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatal(err)
	}

	// Remove frames from an earlier run, so a smaller -steps does not
	// leave old frames behind for the video.
	old, _ := filepath.Glob(filepath.Join(*outDir, "frame_*.png"))
	for _, f := range old {
		if err := os.Remove(f); err != nil {
			log.Fatal(err)
		}
	}

	for i, step := range steps {
		t := float32(step) / totalSteps
		out := image.NewNRGBA(image.Rect(0, 0, w, h))
		for p := 0; p < w*h; p++ {
			for c := 0; c < 3; c++ {
				xt := (1-t)*x0[p*3+c] + t*eps[p*3+c]
				out.Pix[p*4+c] = toByte(xt)
			}
			out.Pix[p*4+3] = 255
		}

		// frame_<order>_step<timestep>.png: the order number keeps the video sorted.
		name := filepath.Join(*outDir, fmt.Sprintf("frame_%04d_step%04d.png", i+1, step))
		if err := save(name, out); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("frame %4d/%d  step %4d  t=%.3f  -> %s\n", i+1, len(steps), step, t, name)
	}
}

// toByte maps a value in [-1, 1] back to 0..255. Values outside are clipped.
func toByte(v float32) uint8 {
	v = (v + 1) / 2 * 255
	switch {
	case v < 0:
		return 0
	case v > 255:
		return 255
	}
	return uint8(v + 0.5)
}

func load(path string) (*image.NRGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	// Convert any PNG type (RGB, RGBA, gray, palette...) to plain 8-bit NRGBA.
	dst := image.NewNRGBA(img.Bounds())
	draw.Draw(dst, dst.Bounds(), img, img.Bounds().Min, draw.Src)
	return dst, nil
}

func save(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
