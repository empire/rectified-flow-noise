# rectified-flow-noise

Add noise to an image the way Stable Diffusion 3 does during training, and turn the result into two videos: image to noise, and noise to image.

SD3 uses rectified flow. Each noisy image is a straight-line blend of the clean image and random noise:

```
x_t = (1 - t) * x0 + t * eps
```

- `x0` is the clean image, with pixel values scaled to [-1, 1].
- `eps` is Gaussian noise with mean 0 and standard deviation 1.
- `t` is the time, from 0 to 1. SD3 uses 1000 steps, so step 500 means `t = 0.5`.

At `t = 0` you have 100% image. At `t = 0.5` you have 50% image and 50% noise. At `t = 1` you have 100% noise.

## Requirements

- Go 1.22 or newer
- `make`
- `ffmpeg` with `libx264` (only for the videos)

## Quick start

Put your image at `sample.png` (PNG), then run:

```sh
make STEPS=20 DURATION=10
```

This writes 20 noisy frames to `noisy/` and two videos, each 10 seconds long:

| File | What it shows |
| --- | --- |
| `image_to_noise.mp4` | The image fades into noise (`t` goes up). |
| `noise_to_image.mp4` | The same frames in reverse order (`t` goes down). |

Each frame is shown for `DURATION / STEPS` seconds. With 20 frames in 10 seconds, that is 0.5 seconds per frame.

## Make variables

| Variable | Default | Meaning |
| --- | --- | --- |
| `STEPS` | `10` | Number of noisy frames, from 1 to 1000. |
| `DURATION` | `10` | Length of each video in seconds. |
| `SEED` | `42` | Random seed. The same seed gives the same noise. |
| `IN` | `sample.png` | Input image. |
| `FRAMES_DIR` | `noisy` | Folder for the frames. |
| `FWD_VIDEO` | `image_to_noise.mp4` | Name of the first video. |
| `REV_VIDEO` | `noise_to_image.mp4` | Name of the second video. |
| `OUT_FPS` | `30` | Frame rate of the video files. |

Targets: `make` (both videos), `make frames`, `make image-to-noise`, `make noise-to-image`, `make clean`.

## Frames only

To skip the videos, run the Go program directly:

```sh
go run . -steps 10
```

Flags: `-in` (default `sample.png`), `-out` (default `noisy`), `-seed` (default `42`), `-steps` (default `10`).

The program spreads the steps evenly over 1 to 1000, so `-steps 10` gives steps 100, 200, … 1000 and `-steps 20` gives 50, 100, … 1000. Frame names look like `frame_0005_step0250.png`: the first number keeps the order, and the second is the SD3 step.

## Notes

- **One noise pattern.** The program makes `eps` once and uses it for every frame. The frames then form one smooth sequence, and the image fades into the same noise pattern.
- **No clean frame.** The first frame is step `1000 / STEPS`, not step 0. Step 0 is the original image, so the videos never show it exactly.
- **Pixels, not latents.** The real SD3 adds this noise to VAE latents. This program adds it to RGB pixels, because pixel noise is easy to see. The formula is the same.
- **Clipping.** Noise with standard deviation 1 often goes outside [-1, 1]. The program clips those values, so frames near `t = 1` look harsh and grey.
- **Old frames.** Each run deletes the old `frame_*.png` files in the output folder first.
- **Video size.** H.264 needs an even width and height, so ffmpeg crops one pixel row from an image with an odd height.
