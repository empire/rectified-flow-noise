# Noise videos from one image, using the SD3 rectified flow formula.
#
#   make                          10 frames, 10 s per video (1 s per frame)
#   make STEPS=20 DURATION=10     20 frames, 10 s per video (0.5 s per frame)
#   make STEPS=50 DURATION=5 SEED=7
#
# It builds two videos from the same frames:
#   image_to_noise.mp4   the image fades into noise  (t goes up)
#   noise_to_image.mp4   the same frames, played backward (t goes down)
#
# Each frame is shown for DURATION / STEPS seconds.

# Number of noisy frames (1..1000).
STEPS      ?= 10
# Length of each video in seconds.
DURATION   ?= 10
# Same seed = same noise.
SEED       ?= 42
IN         ?= sample.png
FRAMES_DIR ?= noisy
FWD_VIDEO  ?= image_to_noise.mp4
REV_VIDEO  ?= noise_to_image.mp4
# Playback frame rate of the video files.
OUT_FPS    ?= 30

REV_DIR := $(FRAMES_DIR)/reverse

# Input rate: frames per second = STEPS / DURATION  (20 / 10 = 2 fps = 0.5 s each)
FPS := $(shell awk 'BEGIN { printf "%.6f", $(STEPS) / $(DURATION) }')

# crop: h264 with yuv420p needs even width and height (this image is 1312x1199).
# -framerate sets how long each frame is shown; -r repeats frames up to OUT_FPS,
# so players get a normal frame rate and each video is exactly DURATION long.
FFMPEG := ffmpeg -y -hide_banner -loglevel warning -framerate $(FPS) -pattern_type glob
ENCODE := -vf "crop=trunc(iw/2)*2:trunc(ih/2)*2" -r $(OUT_FPS) -c:v libx264 -crf 18 -pix_fmt yuv420p

.PHONY: all video frames reverse-frames image-to-noise noise-to-image clean

all: video

video: image-to-noise noise-to-image

# Always regenerate: the frames depend on STEPS and SEED, which make cannot track.
frames:
	go run . -in $(IN) -out $(FRAMES_DIR) -steps $(STEPS) -seed $(SEED)

# Symlinks to the same frames in reverse order (last frame first). This keeps
# memory low for large STEPS, unlike the ffmpeg reverse filter.
reverse-frames: frames
	rm -rf $(REV_DIR) && mkdir -p $(REV_DIR)
	n=1; for f in $$(ls $(FRAMES_DIR)/frame_*.png | sort -r); do \
		ln -s "../$$(basename $$f)" "$(REV_DIR)/rev_$$(printf %04d $$n).png"; \
		n=$$((n + 1)); \
	done

image-to-noise: frames
	$(FFMPEG) -i '$(FRAMES_DIR)/frame_*.png' $(ENCODE) $(FWD_VIDEO)
	@echo "$(FWD_VIDEO): $(STEPS) frames, $(DURATION) s, $$(awk 'BEGIN { printf "%g", $(DURATION) / $(STEPS) }') s per frame"

noise-to-image: reverse-frames
	$(FFMPEG) -i '$(REV_DIR)/rev_*.png' $(ENCODE) $(REV_VIDEO)
	@echo "$(REV_VIDEO): $(STEPS) frames, $(DURATION) s, $$(awk 'BEGIN { printf "%g", $(DURATION) / $(STEPS) }') s per frame"

clean:
	rm -rf $(FRAMES_DIR) $(FWD_VIDEO) $(REV_VIDEO)
