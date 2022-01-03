package utils

import (
	"path"
	"path/filepath"
	"runtime"
)

func RootDir() string {
	_, b, _, _ := runtime.Caller(0)
	d := path.Join(path.Dir(b))
	return filepath.Dir(filepath.Dir(d))
}

func Join(base, file string) string {
	return path.Join(base, file)
}
