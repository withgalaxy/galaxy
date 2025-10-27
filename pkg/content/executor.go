package content

import (
	"github.com/cameron-webmatter/galaxy/pkg/executor"
)

func init() {
	executor.RegisterGlobalFunc("content", "NewCollections", func(args ...interface{}) (interface{}, error) {
		if len(args) < 1 {
			return nil, nil
		}
		contentDir, ok := args[0].(string)
		if !ok {
			return nil, nil
		}
		return NewCollections(contentDir), nil
	})
}
