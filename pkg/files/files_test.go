package files

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestFile(t *testing.T) {
	f, err := os.Open("files.go")
	if err != nil {
		t.Error(err)
	}
	fi, err := f.Stat()
	if err != nil {
		t.Error(err)
	}
	nf := File{
		Name:     fi.Name(),
		Size:     fi.Size(),
		Mode:     fi.Mode(),
		ModTime:  fi.ModTime(),
		IsDir:    fi.IsDir(),
		Parts:    []string{},
		Contents: []File{},
	}
	fmt.Println(nf)
	byt, err := json.Marshal(nf)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(byt))
	var umnf File
	if err := json.Unmarshal(byt, &umnf); err != nil {
		t.Error(err)
	}

	fmt.Println(umnf)
}
