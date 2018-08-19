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

	"github.com/golang/glog"
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

func fileFromFileInfo(f os.FileInfo) File {
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

func UploadDirectory(host, name string, files []File) error {
	filesBytes, err := json.Marshal(files)
	if err != nil {
		return err
	}
	resp, err := http.Post(
		fmt.Sprintf("%s/directory/%s", host, name),
		"application/json",
		bytes.NewBuffer(filesBytes),
	)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusConflict {
		return errors.New("Cannot upload file metadata, there is already an active dir with this name")
	}
	if resp.StatusCode != http.StatusCreated {
		glog.Error(resp.StatusCode)
		return errors.New("directory not created")
	}
	return nil
}

func ReadAllFiles() (files []File, err error) {
	return readDir("./")
}

func readDir(dirName string) (files []File, err error) {
	fmt.Println(dirName)
	filesInfos, err := ioutil.ReadDir(dirName)
	if err != nil {
		return
	}
	for _, fi := range filesInfos {
		fmt.Println(fi.Name())
		file := fileFromFileInfo(fi)

		if fi.IsDir() {
			contents, err := readDir(dirName + fi.Name() + "/")
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
			contents, err := uploadChunks(host, path+f.Name+"/", f.Contents)
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
	const bufSize = 1 << 22
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

func CreateFiles(host string, files []File) error {
	return createFiles(host, "./", files)
}

func createFiles(host, path string, files []File) error {
	for _, file := range files {
		createFile(host, path, file)
		if file.IsDir {
			if err := createFiles(host, path+file.Name+"/", file.Contents); err != nil {
				return err
			}
		}
	}
	return nil
}

func createFile(host, path string, file File) error {
	if file.IsDir {
		if err := os.Mkdir(path+file.Name, file.Mode); err != nil {
			return err
		}
	} else {
		fi, err := os.Create(path + file.Name)
		fi.Chmod(file.Mode)
		if err != nil {
			return err
		}
		for _, sha := range file.Parts {
			resp, err := http.Get(host + "/chunk/" + sha)
			if err != nil {
				return err
			}
			if _, err := io.Copy(fi, resp.Body); err != nil {
				return err
			}
		}
		if err := os.Chtimes(path+file.Name, file.ModTime, file.ModTime); err != nil {
			return err
		}
	}
	return nil
}
