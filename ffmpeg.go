package voices

import (
	"context"
)

func ToMp3(ctx context.Context, in, out string) error {
	return executive(ctx, "ffmpeg", "-y", "-i", in, "-acodec", "libmp3lame", out)
}
