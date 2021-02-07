package voices

import (
	"context"
	"fmt"
	"os/exec"
)

func executive(ctx context.Context, name string, args ...string) error {
	buf, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(buf))
	}
	return nil
}
