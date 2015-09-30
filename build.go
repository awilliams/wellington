package wellington

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fatih/color"
	libsass "github.com/wellington/go-libsass"
)

var testch chan struct{}

// BuildOptions holds a set of read only arguments to the builder.
// Channels from this are used to communicate between the workers
// and loaders executing builds.
type BuildOptions struct {
	wg      sync.WaitGroup
	closing chan struct{}

	workwg sync.WaitGroup
	err    error
	done   chan error
	status chan error
	queue  chan string

	async      bool
	paths      []string
	bArgs      *BuildArgs
	partialMap *SafePartialMap
}

// NewBuild accepts arguments to reate a new Builder
func NewBuild(paths []string, args *BuildArgs, pMap *SafePartialMap, async bool) *BuildOptions {
	return &BuildOptions{
		done:   make(chan error),
		status: make(chan error),

		queue:   make(chan string),
		closing: make(chan struct{}),

		paths:      paths,
		bArgs:      args,
		partialMap: pMap,
		async:      async,
	}
}

// ErrParitalMap when no partial map is found
var ErrPartialMap = errors.New("No partial map found")

// Build compiles all valid Sass files found in the passed paths.
// It will block until all files are compiled.
func (b *BuildOptions) Build() error {

	if b.partialMap == nil {
		return ErrPartialMap
	}

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		b.doBuild()
	}()

	go b.loadWork()

	return <-b.done
}

func (b *BuildOptions) loadWork() {
	paths := pathsToFiles(b.paths, true)
	for _, path := range paths {
		b.queue <- path
	}
	close(b.queue)
}

func (b *BuildOptions) doBuild() {
	for {
		select {
		case <-b.closing:
			return
		case path := <-b.queue:
			if len(path) == 0 {
				b.workwg.Wait()
				b.done <- nil
				return
			}
			b.workwg.Add(1)
			go func(path string) {
				err := b.build(path)
				defer b.workwg.Done()
				if err != nil {
					b.done <- err
				}
			}(path)
		}
	}
}

func (b *BuildOptions) build(path string) error {
	return LoadAndBuild(path, b.bArgs, b.partialMap)
}

// Close shuts down the builder ensuring all go routines have properly
// closed before returning.
func (b *BuildOptions) Close() error {
	close(b.closing)
	b.wg.Wait()
	return nil
}

var inputFileTypes = []string{".scss", ".sass"}

func (b *BuildArgs) getOut(path string) (io.WriteCloser, string, error) {
	var (
		out  io.WriteCloser
		fout string
	)
	if b == nil {
		return nil, "", errors.New("build args is nil")
	}
	if len(b.BuildDir) == 0 {
		out = os.Stdout
		return out, "", nil
	}

	// Build output file based off build directory and input filename
	rel, _ := filepath.Rel(b.Includes, filepath.Dir(path))
	filename := updateFileOutputType(filepath.Base(path))
	fout = filepath.Join(b.BuildDir, rel, filename)

	dir := filepath.Dir(fout)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to create directory: %s",
			dir)
	}

	return out, fout, nil
}

// LoadAndBuild kicks off parser and compiling. It expands directories
// to recursively locate Sass files
// TODO: make this function testable
func LoadAndBuild(path string, gba *BuildArgs, pMap *SafePartialMap) error {
	if len(path) == 0 {
		return errors.New("invalid path passed")
	}

	// file detected!
	if isImportable(path) {
		out, fout, err := gba.getOut(path)
		if err != nil {
			return err
		}
		return loadAndBuild(path, gba, pMap, out, fout)
	}

	out, fout, err := gba.getOut(path)
	if err != nil {
		return err
	}
	err = loadAndBuild(path, gba, pMap, out, fout)
	if err != nil {
		return err
	}

	return nil
}

func loadAndBuild(sassFile string, gba *BuildArgs, partialMap *SafePartialMap, out io.WriteCloser, fout string) error {

	defer func() {
		// Complicated logic to avoid inspecting os.Stdout which would
		// race against testing package
		if closer, ok := out.(*io.PipeWriter); ok {
			fmt.Printf("closing % #v\n", closer)
			closer.Close()
		} else if closer, ok := out.(*os.File); ok {
			closer.Close()
		} else {
			fmt.Printf("failed % #v\n", out)
		}
	}()
	imageDir := gba.Dir
	// If no imagedir specified, assume relative to the input file
	if len(imageDir) == 0 {
		imageDir = filepath.Dir(sassFile)
	}

	ctx := libsass.NewContext()
	ctx.Payload = gba.Payload
	ctx.OutputStyle = gba.Style
	ctx.ImageDir = imageDir
	ctx.FontDir = gba.Font
	// Assumption that output is a file
	ctx.BuildDir = filepath.Dir(fout)
	ctx.GenImgDir = gba.Gen
	ctx.MainFile = sassFile
	ctx.Comments = gba.Comments
	ctx.IncludePaths = []string{filepath.Dir(sassFile)}

	ctx.Imports.Init()
	if gba.Includes != "" {
		ctx.IncludePaths = append(ctx.IncludePaths,
			strings.Split(gba.Includes, ",")...)
	}
	// TODO: remove this!
	fRead, err := os.Open(sassFile)
	if err != nil {
		return err
	}
	defer fRead.Close()
	if fout != "" {
		out, err = os.Create(fout)
		if err != nil {
			return fmt.Errorf("Failed to create file: %s", sassFile)
		}
	}

	err = ctx.FileCompile(sassFile, out)
	if err != nil {
		return errors.New(color.RedString("%s", err))
	}

	for _, inc := range ctx.ResolvedImports {
		partialMap.AddRelation(ctx.MainFile, inc)
	}

	go func(sassFile string) {
		select {
		case <-testch:
		default:
			fmt.Printf("Rebuilt: %s\n", sassFile)
		}
	}(sassFile)
	return nil
}

func updateFileOutputType(filename string) string {
	for _, filetype := range inputFileTypes {
		filename = strings.Replace(filename, filetype, ".css", 1)
	}
	return filename
}
