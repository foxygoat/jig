package serve

import (
	"encoding/json"
	"io/fs"

	"github.com/google/go-jsonnet"
	"google.golang.org/grpc/metadata"
)

type Request struct {
	Metadata metadata.MD
	Method   string
	Input    string
}

type Evaluator interface {
	Evaluate(request Request, vfs fs.FS) (output string, err error)
}

type EvaluatorFunc func(request Request, vfs fs.FS) (output string, err error)

func (ef EvaluatorFunc) Evaluate(request Request, vfs fs.FS) (output string, err error) {
	return ef(request, vfs)
}

func JsonnetEvaluator() Evaluator {
	return EvaluatorFunc(func(request Request, vfs fs.FS) (output string, err error) {
		vm := jsonnet.MakeVM()
		metadataJSON, _ := json.Marshal(request.Metadata)
		vm.TLACode("input", request.Input)
		vm.TLACode("metadata", string(metadataJSON))
		filename := request.Method + ".jsonnet"
		b, err := fs.ReadFile(vfs, filename)
		if err != nil {
			return "", err
		}
		return vm.EvaluateAnonymousSnippet(filename, string(b))
	})
}
