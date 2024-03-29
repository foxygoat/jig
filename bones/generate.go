package bones

import (
	"fmt"
	"os"
	"path"

	"foxygo.at/jig/log"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Generate writes exemplars for the methods specified by targets that match
// the methods in the protoSet file, or all methods if targets is an empty
// slice. Method exemplars are written to stdout if methodDir is empty,
// otherwise each method is written to a separate file in that directory.
// Existing files will not be overwritten unless force is true.
func Generate(logger log.Logger, fds *descriptorpb.FileDescriptorSet, methodDir string, force bool, targets []string, formatOpts *FormatterOptions) error {
	files, err := protodesc.NewFiles(fds)
	if err != nil {
		return err
	}

	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		err = genFile(logger, fd, methodDir, force, targets, formatOpts)
		return err == nil
	})

	return err
}

func genFile(logger log.Logger, fd protoreflect.FileDescriptor, methodDir string, force bool, targets []string, formatOpts *FormatterOptions) error {
	for _, sd := range services(fd) {
		for _, md := range methods(sd) {
			if err := genMethod(logger, md, methodDir, force, targets, formatOpts); err != nil {
				return err
			}
		}
	}
	return nil
}

func genMethod(logger log.Logger, md protoreflect.MethodDescriptor, methodDir string, force bool, targets []string, formatOpts *FormatterOptions) error {
	var err error
	if !match(md, targets) {
		return nil
	}

	formatter := newFormatter(formatOpts)
	f := os.Stdout
	if methodDir != "" {
		flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		if !force {
			// O_EXCL causes os.OpenFile (open(2) syscall) to return
			// os.ErrExist if the file exists.
			flag |= os.O_EXCL
		}
		filename := path.Join(methodDir, string(md.FullName())+formatter.Extension())
		f, err = os.OpenFile(filename, flag, 0666)
		if err != nil {
			if os.IsExist(err) && !force {
				// skip existing files when not forcing
				logger.Debugf("skip existing file %q, use --force to override", md.FullName())
				return nil
			}
			return err
		}
		logger.Debugf("created file %q", filename)
		defer func() {
			cerr := f.Close()
			if err == nil {
				err = cerr
			}
		}()
	}

	logger.Debugf("writing bones of %q", md.FullName())
	_, err = fmt.Fprintln(f, formatter.MethodExemplar(md))
	// Print a separator on stdout to separate multiple exemplars
	if f == os.Stdout {
		fmt.Fprintln(f)
	}
	return err
}

func match(md protoreflect.MethodDescriptor, targets []string) bool {
	if len(targets) == 0 {
		// If no targets, generate all methods
		return true
	}
	pkg := string(md.ParentFile().Package())
	svc := string(md.Parent().Name())
	mthd := string(md.Name())
	matches := map[string]bool{
		svc:              true,
		svc + "." + mthd: true,
		mthd:             true,
	}
	if pkg != "" {
		matches[pkg] = true
		matches[pkg+"."+svc] = true
		matches[pkg+"."+svc+"."+mthd] = true
	}

	for _, target := range targets {
		if matches[target] {
			return true
		}
	}
	return false
}
