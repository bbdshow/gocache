package gocache

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	fileMode = os.FileMode(0666)
)

type Disk struct {
	filename string
}

//  NewDisk
func NewDisk(filename string) *Disk {
	if filename == "" {
		dir, err := currentDir()
		if err != nil {
			panic(fmt.Sprintf("get current dir error %v", err))
		}
		filename = filepath.Join(dir, "gocache")
	} else {
		f, err := filepath.Abs(filename)
		if err != nil {
			panic(fmt.Sprintf("filename abs error %v", err))
		}
		filename = f
	}

	d := Disk{
		filename: filename,
	}
	return &d
}

// WriteToFile 写数据到文件 如果之前文件存在，则删除
func (d *Disk) WriteToFile(data []byte) error {
	if filenameExists(d.filename) {
		if err := os.Remove(d.filename); err != nil {
			return err
		}
	} else {
		dir := filepath.Dir(d.filename)
		if err := os.MkdirAll(dir, fileMode); err != nil {
			return err
		}
	}

	file, err := os.OpenFile(d.filename, os.O_RDWR|os.O_CREATE, fileMode)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	_ = file.Close()
	if err != nil {
		return err
	}
	return nil
}

// ReadFromFile 从文件读取数据
func (d *Disk) ReadFromFile() ([]byte, error) {
	data := make([]byte, 0)
	if !filenameExists(d.filename) {
		return data, nil
	}

	file, err := os.Open(d.filename)
	if err != nil {
		return nil, err
	}
	data, err = ioutil.ReadAll(file)
	_ = file.Close()
	if err != nil {
		return nil, err
	}
	return data, nil
}

// currentDir 当前文件夹
func currentDir() (string, error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0])) //返回绝对路径  filepath.Dir(os.Args[0])去除最后一个元素的路径
	if err != nil {
		return "", err
	}
	return strings.Replace(dir, "\\", "/", -1), nil //将\替换成/
}

// filenameExists 文件是否存在
func filenameExists(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		return false
	}
	return true
}
