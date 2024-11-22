//go:build windows

package wclayer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Microsoft/go-winio"
	"github.com/Microsoft/hcsshim/internal/hcserror"
	"github.com/Microsoft/hcsshim/internal/oc"
	"github.com/Microsoft/hcsshim/internal/safefile"
	"go.opencensus.io/trace"
)

// ImportLayer will take the contents of the folder at importFolderPath and import
// that into a layer with the id layerId.  Note that in order to correctly populate
// the layer and interperet the transport format, all parent layers must already
// be present on the system at the paths provided in parentLayerPaths.
func ImportLayer(ctx context.Context, path string, importFolderPath string, parentLayerPaths []string) (err error) {
	title := "hcsshim::ImportLayer"
	ctx, span := oc.StartSpan(ctx, title)
	defer span.End()
	defer func() { oc.SetSpanStatus(span, err) }()
	span.AddAttributes(
		trace.StringAttribute("path", path),
		trace.StringAttribute("importFolderPath", importFolderPath),
		trace.StringAttribute("parentLayerPaths", strings.Join(parentLayerPaths, ", ")))

	// Generate layer descriptors
	layers, err := layerPathsToDescriptors(ctx, parentLayerPaths)
	if err != nil {
		return err
	}

	err = importLayer(&stdDriverInfo, path, importFolderPath, layers)
	if err != nil {
		return hcserror.New(err, title, "")
	}
	return nil
}

// LayerWriter is an interface that supports writing a new container image layer.
type LayerWriter interface {
	// Add adds a file to the layer with given metadata.
	Add(name string, fileInfo *winio.FileBasicInfo) error
	// AddLink adds a hard link to the layer. The target must already have been added.
	AddLink(name string, target string) error
	// Remove removes a file that was present in a parent layer from the layer.
	Remove(name string) error
	// Write writes data to the current file. The data must be in the format of a Win32
	// backup stream.
	Write(b []byte) (int, error)
	// Close finishes the layer writing process and releases any resources.
	Close() error
}

type legacyLayerWriterWrapper struct {
	ctx context.Context
	s   *trace.Span

	*legacyLayerWriter
	path             string
	parentLayerPaths []string
}

func (r *legacyLayerWriterWrapper) Close() (err error) {
	defer func() {
		fmt.Fprintf(os.Stderr, "MEOW: defer s.End\n")
		r.s.End()
	}()
	defer func() {
		fmt.Fprintf(os.Stderr, "MEOW: defer setspanstatus\n")
		oc.SetSpanStatus(r.s, err)
	}()
	defer func() {
		fmt.Fprintf(os.Stderr, "MEOW: defer removeall\n")
		os.RemoveAll(r.root.Name())
	}()
	defer func() {
		fmt.Fprintf(os.Stderr, "MEOW: defer closeRoots\n")
		r.legacyLayerWriter.CloseRoots()
	}()

	fmt.Fprintf(os.Stderr, "MEOW: calling legacyLayerwriterWrapper.legacyLayerWriter.Close()\n")
	err = r.legacyLayerWriter.Close()
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "MEOW: Calling ImportLayer\n")
	if err = ImportLayer(r.ctx, r.destRoot.Name(), r.path, r.parentLayerPaths); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "MEOW: Looping Toombstones\n")
	for _, name := range r.Tombstones {
		fmt.Fprintf(os.Stderr, "MEOW: Tombstone-RemoveRelative\n")
		if err = safefile.RemoveRelative(name, r.destRoot); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	// Add any hard links that were collected.
	fmt.Fprintf(os.Stderr, "MEOW: Looping Links\n")
	for _, lnk := range r.PendingLinks {
		fmt.Fprintf(os.Stderr, "MEOW: Link-RemoveRelative\n")
		if err = safefile.RemoveRelative(lnk.Path, r.destRoot); err != nil && !os.IsNotExist(err) {
			return err
		}
		fmt.Fprintf(os.Stderr, "MEOW: Link-LinkRelative\n")
		if err = safefile.LinkRelative(lnk.Target, lnk.TargetRoot, lnk.Path, r.destRoot); err != nil {
			return err
		}
	}

	// The reapplyDirectoryTimes must be called AFTER we are done with Tombstone
	// deletion and hard link creation. This is because Tombstone deletion and hard link
	// creation updates the directory last write timestamps so that will change the
	// timestamps added by the `Add` call. Some container applications depend on the
	// correctness of these timestamps and so we should change the timestamps back to
	// the original value (i.e the value provided in the Add call) after this
	// processing is done.
	fmt.Fprintf(os.Stderr, "MEOW: reapplyDirectoryTimes\n")
	err = reapplyDirectoryTimes(r.destRoot, r.changedDi)
	if err != nil {
		return err
	}

	// Prepare the utility VM for use if one is present in the layer.
	if r.HasUtilityVM {
		fmt.Fprintf(os.Stderr, "MEOW: EnsureNotReparsePointRelative\n")
		err := safefile.EnsureNotReparsePointRelative("UtilityVM", r.destRoot)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "MEOW: ProcessUtilityVMImage\n")
		err = ProcessUtilityVMImage(r.ctx, filepath.Join(r.destRoot.Name(), "UtilityVM"))
		if err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "MEOW: DONE calling legacyLayerwriterWrapper.Close()\n")
	return nil
}

// NewLayerWriter returns a new layer writer for creating a layer on disk.
// The caller must have taken the SeBackupPrivilege and SeRestorePrivilege privileges
// to call this and any methods on the resulting LayerWriter.
func NewLayerWriter(ctx context.Context, path string, parentLayerPaths []string) (_ LayerWriter, err error) {
	ctx, span := oc.StartSpan(ctx, "hcsshim::NewLayerWriter")
	defer func() {
		if err != nil {
			oc.SetSpanStatus(span, err)
			span.End()
		}
	}()
	span.AddAttributes(
		trace.StringAttribute("path", path),
		trace.StringAttribute("parentLayerPaths", strings.Join(parentLayerPaths, ", ")))

	if len(parentLayerPaths) == 0 {
		// This is a base layer. It gets imported differently.
		f, err := safefile.OpenRoot(path)
		if err != nil {
			return nil, err
		}
		return &baseLayerWriter{
			ctx:  ctx,
			s:    span,
			root: f,
		}, nil
	}

	importPath, err := os.MkdirTemp("", "hcs")
	if err != nil {
		return nil, err
	}
	w, err := newLegacyLayerWriter(importPath, parentLayerPaths, path)
	if err != nil {
		return nil, err
	}
	return &legacyLayerWriterWrapper{
		ctx:               ctx,
		s:                 span,
		legacyLayerWriter: w,
		path:              importPath,
		parentLayerPaths:  parentLayerPaths,
	}, nil
}
