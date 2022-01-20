package serve

import (
	"io/fs"

	"github.com/google/go-jsonnet"
)

type Evaluator interface {
	Evaluate(method, input string, vfs fs.FS) (output string, err error)
}

type EvaluatorFunc func(method, input string, vfs fs.FS) (output string, err error)

func (ef EvaluatorFunc) Evaluate(method, input string, vfs fs.FS) (output string, err error) {
	return ef(method, input, vfs)
}

func JsonnetEvaluator() Evaluator {
	return EvaluatorFunc(func(method, input string, vfs fs.FS) (output string, err error) {
		vm := jsonnet.MakeVM()
		vm.TLACode("input", input)
		filename := method + ".jsonnet"
		b, err := fs.ReadFile(vfs, filename)
		if err != nil {
			return "", err
		}
		return vm.EvaluateAnonymousSnippet(filename, string(b))
	})
}
