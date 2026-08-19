package fsutil

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
)

// ReserveOrdinal atomically reserves the next ordinal for a document family
// whose files carry a monotonically increasing creation number (handoffs,
// checkpoints). The O_EXCL reservation file in reservationDir makes
// concurrent CLI processes retry with the next value instead of assigning
// the same one. Releasing the returned func removes the marker; a crash can
// leave it behind, intentionally consuming that ordinal so it can never be
// reused.
//
// highest reports the largest ordinal already committed to disk by the
// caller's family; inUse reports whether one specific ordinal is committed.
// The second call closes the scan-then-reserve race: another caller may
// commit and release an ordinal after highest's snapshot but before we
// create its now-vacant reservation marker.
func ReserveOrdinal(reservationDir string, highest func() (uint64, error), inUse func(uint64) (bool, error)) (uint64, func(), error) {
	if err := os.MkdirAll(reservationDir, 0o755); err != nil {
		return 0, nil, err
	}
	for {
		ordinal, err := nextOrdinal(reservationDir, highest)
		if err != nil {
			return 0, nil, err
		}
		reservation := filepath.Join(reservationDir, strconv.FormatUint(ordinal, 10))
		file, err := os.OpenFile(reservation, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return 0, nil, err
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(reservation)
			return 0, nil, err
		}
		taken, err := inUse(ordinal)
		if err != nil {
			_ = os.Remove(reservation)
			return 0, nil, err
		}
		if taken {
			_ = os.Remove(reservation)
			continue
		}
		return ordinal, func() { _ = os.Remove(reservation) }, nil
	}
}

// nextOrdinal returns one past the largest ordinal seen on disk. Active or
// crash-left reservation files participate in the maximum, preventing reuse.
func nextOrdinal(reservationDir string, highest func() (uint64, error)) (uint64, error) {
	max, err := highest()
	if err != nil {
		return 0, err
	}
	reservations, err := os.ReadDir(reservationDir)
	if err != nil && !os.IsNotExist(err) {
		return 0, err
	}
	for _, reservation := range reservations {
		if reservation.IsDir() {
			continue
		}
		if ordinal, err := strconv.ParseUint(reservation.Name(), 10, 64); err == nil && ordinal > max {
			max = ordinal
		}
	}
	if max == ^uint64(0) {
		return 0, errors.New("ordinal exhausted")
	}
	return max + 1, nil
}
