package replaygain

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.senan.xyz/wrtag/addon"
	"go.senan.xyz/wrtag/replaygain"
	"go.senan.xyz/wrtag/tags"
)

func init() {
	addon.Register("replaygain", NewReplayGainAddon)
}

type ReplayGainAddon struct {
	truePeak bool
	force    bool
}

func NewReplayGainAddon(conf string) (ReplayGainAddon, error) {
	var a ReplayGainAddon
	for _, arg := range strings.Fields(conf) {
		switch arg {
		case "true-peak":
			a.truePeak = true
		case "force":
			a.force = true
		}
	}
	return a, nil
}

func (a ReplayGainAddon) ProcessRelease(ctx context.Context, paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	if !a.force {
		first, err := tags.ReadTags(paths[0])
		if err != nil {
			return fmt.Errorf("read first file: %w", err)
		}
		if first.Get(tags.ReplayGainTrackGain) != "" {
			return nil
		}
	}

	albumLev, trackLevs, err := replaygain.Calculate(ctx, a.truePeak, paths)
	if err != nil {
		return fmt.Errorf("calculate: %w", err)
	}

	var trackErrs []error
	for i := range paths {
		trackL, path := trackLevs[i], paths[i]

		t := tags.NewTags(
			tags.ReplayGainTrackGain, fmtdB(trackL.GaindB),
			tags.ReplayGainTrackPeak, fmtFloat(trackL.Peak, 6),
			tags.ReplayGainAlbumGain, fmtdB(albumLev.GaindB),
			tags.ReplayGainAlbumPeak, fmtFloat(albumLev.Peak, 6),
		)
		if err := tags.WriteTags(path, t); err != nil {
			trackErrs = append(trackErrs, err)
			continue
		}
	}

	return errors.Join(trackErrs...)
}

func (a ReplayGainAddon) String() string {
	return fmt.Sprintf("replaygain (force: %t, true peak: %t)", a.force, a.truePeak)
}

func fmtFloat(v float64, p int) string {
	return strconv.FormatFloat(v, 'f', p, 64)
}
func fmtdB(v float64) string {
	return fmt.Sprintf("%.2f dB", v)
}
