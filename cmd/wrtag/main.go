package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"go.senan.xyz/wrtag/musicbrainz"
	"go.senan.xyz/wrtag/release"
	"go.senan.xyz/wrtag/tags/tagcommon"
	"go.senan.xyz/wrtag/tags/taglib"

	"github.com/peterbourgon/ff/v4"
)

func main() {
	fs := ff.NewFlagSet("wrtag")
	_ = fs.StringLong("path-format", "", "path format")
	_ = fs.StringLong("config", "", "config file (optional)")

	userConfig, _ := os.UserConfigDir()
	configPath := filepath.Join(userConfig, "wrtag", "config")

	ffopt := []ff.Option{
		ff.WithEnvVarPrefix("WRTAG"),
		ff.WithConfigFileFlag("config"),
	}
	if stat, err := os.Stat(configPath); err == nil && stat.Mode().IsRegular() {
		ffopt = append(ffopt,
			ff.WithConfigFile(configPath),
			ff.WithConfigFileParser(ff.PlainParser),
		)
	}
	if err := ff.Parse(fs, os.Args[1:], ffopt...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	tg := taglib.TagLib{}
	mb := musicbrainz.NewClient()

	for _, dir := range fs.GetArgs() {
		if err := processDir(tg, mb, dir); err != nil {
			log.Printf("error processing dir %q: %v", dir, err)
		}
	}
}

func processDir(tg taglib.TagLib, mb *musicbrainz.Client, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var files []tagcommon.File
	for _, entry := range entries {
		if path := filepath.Join(dir, entry.Name()); tg.CanRead(path) {
			file, err := tg.Read(path)
			if err != nil {
				return fmt.Errorf("read track: %w", err)
			}
			files = append(files, file)
		}
	}
	if len(files) == 0 {
		return fmt.Errorf("no tracks in dir")
	}
	sort.SliceStable(files, func(i, j int) bool {
		return files[i].TrackNumber() < files[j].TrackNumber()
	})

	var query musicbrainz.Query
	{
		f := files[0] // search with first file only
		query.MBReleaseID = f.MBReleaseID()
		query.MBArtistID = first(f.MBArtistID())
		query.MBReleaseGroupID = f.MBReleaseGroupID()
		query.Release = f.Album()
		query.Artist = f.AlbumArtist()
		query.Format = f.MediaFormat()
		query.Date = f.Date()
		query.Label = f.Label()
		query.CatalogueNum = f.CatalogueNum()
		query.NumTracks = len(files)
	}

	score, resp, err := mb.SearchRelease(context.Background(), query)
	if err != nil {
		return fmt.Errorf("search release: %w", err)
	}
	if score < 100 {
		return fmt.Errorf("score too low")
	}

	releaseTags := release.FromTags(files)
	releaseMB := release.FromMusicBrainz(resp)
	if len(releaseTags.Tracks) != len(releaseMB.Tracks) {
		return fmt.Errorf("track count mismatch %d/%d", len(releaseTags.Tracks), len(releaseMB.Tracks))
	}

	fmt.Println()
	fmt.Printf("dir: %q\n", dir)
	fmt.Print(release.Diff(releaseTags, releaseMB))

	release.ToTags(releaseMB, files)

	var errs []error
	for _, t := range files {
		errs = append(errs, t.Close())
	}

	return errors.Join(errs...)
}

func first[T comparable](is []T) T {
	var z T
	for _, i := range is {
		if i != z {
			return i
		}
	}
	return z
}
