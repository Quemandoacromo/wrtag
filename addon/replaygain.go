package addon

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.senan.xyz/wrtag/addon/replaygain"
	"go.senan.xyz/wrtag/tags"
)

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
		existingTag, err := func() (string, error) {
			f, err := tags.ReadTags(paths[0])
			if err != nil {
				return "", err
			}
			return f.Read(tags.ReplayGainTrackGain), nil
		}()
		if err != nil {
			return fmt.Errorf("read first file: %w", err)
		}
		if existingTag != "" {
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
		err := tags.UpdateTags(path, func(t tags.Tags) {
			t.Write(tags.ReplayGainTrackGain, fmtdB(trackL.GaindB))
			t.WriteFloat(tags.ReplayGainTrackPeak, trackL.Peak)
			t.Write(tags.ReplayGainAlbumGain, fmtdB(albumLev.GaindB))
			t.WriteFloat(tags.ReplayGainAlbumPeak, albumLev.Peak)
		})
		if err != nil {
			trackErrs = append(trackErrs, err)
			continue
		}
	}
	return errors.Join(trackErrs...)
}

func (a ReplayGainAddon) String() string {
	return fmt.Sprintf("replaygain (force: %t, true peak: %t)", a.force, a.truePeak)
}

func fmtdB(v float64) string {
	return fmt.Sprintf("%.2f dB", v)
}
