package gocache

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	fileMode = os.FileMode(0666)
)

type Disk struct {
	Filename string
}

//  NewDisk  filename
func NewDisk(filename string) *Disk {

	if filename == "" {
		dir, err := GetCurrentDir()
		if err != nil {
			panic("get current dir err " + err.Error())
		}

		filename = filepath.Join(dir, "data.cache")
	} else {
		f, err := filepath.Abs(filename)
		if err != nil {
			panic("filename abs err " + err.Error())
		}
		filename = f
	}

	d := Disk{
		Filename: filename,
	}

	return &d
}

// WriteToFile 如果之前文件存在，则删除
func (d *Disk) WriteToFile(data []byte) error {
	if FilenameExists(d.Filename) {
		if err := os.Remove(d.Filename); err != nil {
			return err
		}
	} else {
		dir := filepath.Dir(d.Filename)
		if err := os.MkdirAll(dir, fileMode); err != nil {
			return err
		}
	}

	file, err := os.OpenFile(d.Filename, os.O_RDWR|os.O_CREATE, fileMode)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}

// ReadFromFile
func (d *Disk) ReadFromFile() ([]byte, error) {
	data := make([]byte, 0)
	if !FilenameExists(d.Filename) {
		return data, nil
	}

	file, err := os.Open(d.Filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err = ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func GetCurrentDir() (string, error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0])) //返回绝对路径  filepath.Dir(os.Args[0])去除最后一个元素的路径
	if err != nil {
		return "", err
	}
	return strings.Replace(dir, "\\", "/", -1), nil //将\替换成/
}

func FilenameExists(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		return false
	}
	return true
}
