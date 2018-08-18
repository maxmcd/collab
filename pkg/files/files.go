package files

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type File struct {
	Name     string
	Size     int64
	Mode     os.FileMode
	ModTime  time.Time
	IsDir    bool
	Parts    []string
	Contents []File
}

func fileFromFile(f os.FileInfo) File {
	return File{
		Name:     f.Name(),
		Size:     f.Size(),
		Mode:     f.Mode(),
		ModTime:  f.ModTime(),
		IsDir:    f.IsDir(),
		Parts:    []string{},
		Contents: []File{},
	}
}

func Upload(host, file string) {

}

func ReadAllFiles() (files []File, err error) {
	return readDir("./")
}

func readDir(dirName string) (files []File, err error) {
	fmt.Println(dirName)
	fls, err := ioutil.ReadDir(dirName)
	if err != nil {
		return
	}
	for _, f := range fls {
		fmt.Println(f.Name())
		file := fileFromFile(f)

		if f.IsDir() {
			contents, err := readDir(dirName + f.Name())
			if err != nil {
				return files, err
			}
			file.Contents = contents
		}
		files = append(files, file)
	}
	return
}

func UploadChunks(host string, files []File) ([]File, error) {
	return uploadChunks(host, "./", files)
}

func uploadChunks(host, path string, files []File) ([]File, error) {
	var out []File
	fmt.Println(path)
	for _, f := range files {
		if f.IsDir {
			contents, err := uploadChunks(host, path+f.Name, f.Contents)
			if err != nil {
				return files, err
			}
			f.Contents = contents
		} else {
			parts, err := uploadFile(host, path, &f)
			if err != nil {
				return files, err
			}
			f.Parts = parts
		}
		out = append(out, f)
	}
	return out, nil
}

func uploadFile(host, path string, file *File) (hashes []string, err error) {
	const bufSize = 4000
	f, err := os.Open(path + "/" + file.Name)
	if err != nil {
		return
	}
	defer f.Close()
	buffer := make([]byte, bufSize)
	for {
		bytesread, err := f.Read(buffer)
		if err != nil {
			if err != io.EOF {
				return hashes, err
			}
			break
		}
		sum := sha256.Sum256(buffer[:bytesread])
		sha := fmt.Sprintf("%x", sum)
		if err := uploadChunk(host, sha, buffer[:bytesread]); err != nil {
			return hashes, err
		}
		hashes = append(hashes, sha)
	}
	return
}

func checkChunk(host, sha string) (exists bool, err error) {
	resp, err := http.Head(fmt.Sprintf("%s/chunk/%s", host, sha))
	if err != nil {
		return
	}
	exists = resp.StatusCode == http.StatusOK
	return
}

func uploadChunk(host, sha string, body []byte) error {
	exists, err := checkChunk(host, sha)
	if err != nil {
		return err
	}
	if exists == false {
		resp, err := http.Post(fmt.Sprintf("%s/chunk/%s", host, sha), "", bytes.NewBuffer(body))
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusInternalServerError {
			return errors.New("server error uploading chunk")
		}
	}
	return nil
}

func FetchAllFiles(host, name string) (files []File, err error) {
	resp, err := http.Get(fmt.Sprintf("%s/directory/%s", host, name))
	if err != nil {
		return
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if err := json.Unmarshal(bytes, &files); err != nil {
		return files, err
	}

	return files, nil
}
