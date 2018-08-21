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
	Contents []*File
}

func fileFromFileInfo(f os.FileInfo) File {
	return File{
		Name:     f.Name(),
		Size:     f.Size(),
		Mode:     f.Mode(),
		ModTime:  f.ModTime(),
		IsDir:    f.IsDir(),
		Parts:    []string{},
		Contents: []*File{},
	}
}

type Metadata struct {
	Files   []*File
	fileMap map[string]*File
	Host    string
	Name    string
}

func New(host, name string) Metadata {
	return Metadata{
		Files:   []*File{},
		Host:    host,
		Name:    name,
		fileMap: make(map[string]*File),
	}
}

func (md *Metadata) UploadDirectory() error {
	filesBytes, err := json.Marshal(md.Files)
	if err != nil {
		return err
	}
	resp, err := http.Post(
		fmt.Sprintf("%s/directory/%s", md.Host, md.Name),
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

func (md *Metadata) ReadAllFiles() (err error) {
	files, err := readDir("./")
	md.Files = files
	return err
}

func readDir(dirName string) (files []*File, err error) {
	filesInfos, err := ioutil.ReadDir(dirName)
	if err != nil {
		return
	}
	for _, fi := range filesInfos {
		file := fileFromFileInfo(fi)

		if fi.IsDir() {
			contents, err := readDir(dirName + fi.Name() + "/")
			if err != nil {
				return files, err
			}
			file.Contents = contents
		}
		files = append(files, &file)
	}
	return
}

func readFileAndUpload(host, location string) (file *File, err error) {
	fi, err := os.Stat(location)
	if err != nil {
		return
	}
	f := fileFromFileInfo(fi)
	hashes, err := uploadFile(host, location)
	if err != nil {
		return
	}
	f.Parts = hashes
	return &f, err
}

func (md *Metadata) UploadChunks() error {
	_, err := uploadChunks(md.Host, "./", md.Files)
	return err
}

func uploadChunks(host, path string, files []*File) ([]*File, error) {
	for _, f := range files {
		if f.IsDir {
			contents, err := uploadChunks(host, path+f.Name+"/", f.Contents)
			if err != nil {
				return files, err
			}
			f.Contents = contents
		} else {
			parts, err := uploadFile(host, path+"/"+f.Name)
			if err != nil {
				return files, err
			}
			f.Parts = parts
		}
	}
	return files, nil
}

func uploadFile(host, location string) (hashes []string, err error) {
	const bufSize = 1 << 22
	f, err := os.Open(location)
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

func (md *Metadata) FetchAllFiles() error {
	resp, err := http.Get(fmt.Sprintf("%s/directory/%s", md.Host, md.Name))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		return errors.New("Couldn't find a hosted directory by that name. Is the host live?")
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bytes, &md.Files); err != nil {
		return err
	}
	return nil
}

func (md *Metadata) CreateFiles() error {
	return createFiles(md.Host, "./", md.Files)
}

func createFiles(host, path string, files []*File) error {
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

func createFile(host, path string, file *File) error {
	if file.IsDir {
		if err := os.Mkdir(path+file.Name, file.Mode); err != nil {
			return err
		}
	} else {
		return _createFile(host, path+file.Name, file)
	}
	return nil
}

func writeFileBody(host string, to interface{}, file *File) error {
	for _, sha := range file.Parts {
		resp, err := http.Get(host + "/chunk/" + sha)
		if err != nil {
			return err
		}
		if _, err := io.Copy(to.(io.Writer), resp.Body); err != nil {
			return err
		}
	}
	return nil
}

func _createFile(host, name string, file *File) error {
	fi, err := os.Create(name)
	fi.Chmod(file.Mode)
	if err != nil {
		return err
	}
	if err := writeFileBody(host, fi, file); err != nil {
		return err
	}
	if err := os.Chtimes(name, file.ModTime, file.ModTime); err != nil {
		return err
	}
	return nil
}
