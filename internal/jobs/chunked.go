package jobs

import (
	"context"
	"fmt"
)

// Chunked runs op on consecutive chunks of ids, honoring ctx cancellation
// between chunks and emitting Update progress with the noun template
// ("deleting", "applying implication", ...). The returned processed count
// is the number of ids reached when the loop exits (cancelled or
// completed). cancelled is true when ctx tripped before the slice ran out.
func Chunked(ctx context.Context, mgr *Manager, ids []int64, chunkSize int, noun string,
	op func(chunk []int64) error,
) (processed int, cancelled bool, err error) {
	total := len(ids)
	if mgr != nil {
		mgr.Update(0, total, fmt.Sprintf("%s…", noun))
	}
	for start := 0; start < total; start += chunkSize {
		if ctx.Err() != nil {
			return processed, true, nil
		}
		end := min(start+chunkSize, total)
		chunk := ids[start:end]
		if err := op(chunk); err != nil {
			return processed, false, err
		}
		processed = end
		if mgr != nil {
			mgr.Update(processed, total, fmt.Sprintf("%s…", noun))
		}
	}
	return processed, false, nil
}
