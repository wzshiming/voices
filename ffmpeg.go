package voices

import (
	"context"
)

func ToMp3(ctx context.Context, in, out string) error {
	return executive(ctx, "ffmpeg", "-y", "-i", in, "-acodec", "libmp3lame", out)
}

func PlayMp3(ctx context.Context, filename string) error {
	return executive(ctx, "ffplay", "-v", "quiet", "-nodisp", "-autoexit", "-i", filename)
}
