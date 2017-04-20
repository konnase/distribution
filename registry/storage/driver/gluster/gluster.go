package gluster

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	//"io/ioutil"
	"os"
	"path"
	"strings"
	//"reflect"
	//"strconv"
	"time"

	"github.com/docker/distribution/context"
	storagedriver "github.com/docker/distribution/registry/storage/driver"
	"github.com/docker/distribution/registry/storage/driver/base"
	"github.com/docker/distribution/registry/storage/driver/factory"
	"github.com/gluster/gogfapi/gfapi"
	//"github.com/mitchellh/mapstructure"
)

const (
	driverName  = "gluster"
	defaultPath = "/mnt/gluster"
)

type Parameters struct {
	host           string //gluster server的hostname/ip
	volname        string //准备链接的volume的名称
	mountDirectory string //volume挂载的本地目录
}

func init() {
	factory.Register(driverName, &glusterDriverFactory{})
}

//待实现的DriverFactory接口
type glusterDriverFactory struct{}

func (g *glusterDriverFactory) Create(parameters map[string]interface{}) (storagedriver.StorageDriver, error) {
	fmt.Printf("DriverFactory:%v\n", parameters)
	return FromParameters(parameters)
}

//用于实现storagedriver的结构，实现方法在后面
type driver struct {
	vol      *gfapi.Volume //一个gluster storagedriver的卷指针
	mountDir string        //volume挂载的本地目录
}

type baseEmbed struct {
	base.Base
}

// Driver is a storagedriver.StorageDriver implementation backed by a local
// filesystem. All provided paths will be subpaths of the RootDirectory.
type Driver struct {
	baseEmbed
}

func FromParameters(parameters map[string]interface{}) (*Driver, error) {
	fmt.Printf("Before Decode:%v\n", parameters)
	params := Parameters{}
	//Decode方法将parameters解码成params结构，并存储在params中
	// if err := mapstructure.Decode(parameters, &params); err != nil {
	// 	return nil, err
	// }
	params.host = parameters["host"].(string)
	params.volname = parameters["volname"].(string)
	params.mountDirectory = parameters["mountDirectory"].(string)
	fmt.Printf("After Decode:%v\n", params)
	if params.host == "" {
		return nil, fmt.Errorf("No hostname parameter provided!")
	}
	if params.volname == "" {
		return nil, fmt.Errorf("No volname parameter provided!")
	}
	if params.mountDirectory == "" {
		return nil, fmt.Errorf("No mountDirectory parameter provided!")
	}
	return New(params)
}

func New(params Parameters) (*Driver, error) {
	v := new(gfapi.Volume)
	if v == nil {
		return nil, fmt.Errorf("Failed to create volume!")
	}
	if ret := v.Init(params.host, params.volname); ret != 0 {
		return nil, fmt.Errorf("Failed to initialize volume %v!", params.volname)
	}
	fmt.Printf("Volume initialized! Host:%s Volname:%s\n", params.host, params.volname)
	if ret := v.Mount(); ret != 0 {
		return nil, fmt.Errorf("Failed to mount volume %v!", params.volname)
	}
	fmt.Printf("Volume Mounted\n")
	glusterDriver := &driver{
		vol:      v,
		mountDir: params.mountDirectory,
	}
	return &Driver{
		baseEmbed: baseEmbed{
			Base: base.Base{
				StorageDriver: glusterDriver,
			},
		},
	}, nil
}

//implement storagedriver.StorageDriver接口
func (d *driver) Name() string {
	return driverName
}

// GetContent retrieves the content stored at "path" as a []byte.
//该方法里面的ioutil.ReadAll(rc)执行后不返回值，有待解决   暂时改为读固定大小的字节数：rc.Read(p)
func (d *driver) GetContent(ctx context.Context, path string) ([]byte, error) {
	fmt.Println("Begin to call GetContent ")
	rc, err := d.Reader(ctx, path, 0)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	fmt.Println("reader has been executed!")
	var p0 [204800]byte
	p := p0[:]
	_, err = rc.Read(p)
	//p, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	fmt.Println("GetContent has been executed!")
	return p, nil
}

// PutContent stores the []byte content at a location designated by "path".
func (d *driver) PutContent(ctx context.Context, subPath string, contents []byte) error {
	writer, err := d.Writer(ctx, subPath, false)
	if err != nil {
		return err
	}
	defer writer.Close()
	_, err = io.Copy(writer, bytes.NewReader(contents))
	if err != nil {
		writer.Cancel()
		return err
	}
	return writer.Commit()
}

// Reader retrieves an io.ReadCloser for the content stored at "path" with a
// given byte offset.
func (d *driver) Reader(ctx context.Context, path string, offset int64) (io.ReadCloser, error) {
	fmt.Println("called in gluster reader")
	fmt.Printf("read path:%s\n", path)
	file, err := d.vol.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, storagedriver.PathNotFoundError{Path: path}
		}

		return nil, err
	}

	seekPos, err := file.Seek(int64(offset), os.SEEK_SET)
	if err != nil {
		file.Close()
		return nil, err
	} else if seekPos < int64(offset) {
		file.Close()
		return nil, storagedriver.InvalidOffsetError{Path: path, Offset: offset}
	}
	fmt.Println("called in gluster reader finished!")
	return file, nil
}

func (d *driver) Writer(ctx context.Context, fullPath string, append bool) (storagedriver.FileWriter, error) {
	fmt.Println("called in gluster writer")
	fmt.Printf("write path:%s\n", fullPath)
	//fullPath := d.getFullPath(subPath)
	parentDirInit := path.Dir(fullPath) //获取 fullPath 中最后一个分隔符之前的部分（不包含分隔符）
	//fmt.Println(parentDir)
	parentDir := strings.TrimLeft(parentDirInit, "/")
	fmt.Println(parentDir)
	if parentDir != "" {
		if err := d.vol.MkdirAll(parentDir, 0777); err != nil {
			return nil, err
		}

	}
	fmt.Println("No error after MkdirAll")
	fp, err := d.vol.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	var offset int64
	//如果不是向文件追加写，则将文件的大小清零
	if !append {
		err := fp.Truncate(0)
		if err != nil {
			fp.Close()
			return nil, err
		}
	} else {
		n, err := fp.Seek(0, os.SEEK_END) //返回下一次写的偏移量：文件尾    SEEK_SET    SEEK_CUR
		if err != nil {
			fp.Close()
			return nil, err
		}
		offset = int64(n)
	}
	fmt.Println("gluster writer called finished")
	//返回入口参数是文件fp和偏移offset的FileWriter
	return d.newFileWriter(fp, offset), nil
}

// Stat retrieves the FileInfo for the given path, including the current size
// in bytes and the creation time.
func (d *driver) Stat(ctx context.Context, fullPath string) (storagedriver.FileInfo, error) {
	//fullPath := d.fullPath(subPath)
	//fmt.Println("Begin to call Stat")
	//fmt.Println(fullPath)
	fi, err := d.vol.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, storagedriver.PathNotFoundError{Path: fullPath}
		}

		return nil, err
	}
	//fmt.Println("Stat called finished")
	return fileInfo{
		path:     fullPath,
		FileInfo: fi,
	}, nil
}

// List returns a list of the objects that are direct descendants of the given
// path.
func (d *driver) List(ctx context.Context, subPath string) ([]string, error) {
	fmt.Println("Begin to call List")
	fullPath := d.getFullPath(subPath)
	dir, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, storagedriver.PathNotFoundError{Path: subPath}
		}
		return nil, err
	}

	defer dir.Close()

	fileNames, err := dir.Readdirnames(0)
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(fileNames))
	for _, fileName := range fileNames {
		keys = append(keys, path.Join(subPath, fileName))
	}
	fmt.Println("List called finished")
	return keys, nil
}

// Move moves an object stored at sourcePath to destPath, removing the original
// object.
//把文件移动到根目录下怎么表示根目录
func (d *driver) Move(ctx context.Context, sourcePath string, destPath string) error {
	// source := d.getFullPath(sourcePath)
	// dest := d.getFullPath(destPath)
	fmt.Println("Begin to execute Move!")
	if _, err := d.vol.Stat(sourcePath); os.IsNotExist(err) {
		return storagedriver.PathNotFoundError{Path: sourcePath}
	}
	destInit := path.Dir(destPath)
	dest := strings.TrimLeft(destInit, "/")
	fmt.Println("dest genarated!")
	if dest != "" {
		if err := d.vol.MkdirAll(dest, 0755); err != nil {
			return err
		}
	}
	fmt.Println("DestDirectory created!")
	err := d.vol.Rename(sourcePath, destPath)
	return err
}

// Delete recursively deletes all objects stored at "path" and its subpaths.
func (d *driver) Delete(ctx context.Context, subPath string) error {
	fmt.Println("Begin to call Delete")
	_, err := d.vol.Stat(subPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else if err != nil {
		return storagedriver.PathNotFoundError{Path: subPath}
	}
	fullPath := d.getFullPath(subPath)
	err = os.RemoveAll(fullPath)
	fmt.Println("Delete called finished")
	return err
}

// URLFor returns a URL which may be used to retrieve the content stored at the given path.
// May return an UnsupportedMethodErr in certain StorageDriver implementations.
func (d *driver) URLFor(ctx context.Context, path string, options map[string]interface{}) (string, error) {
	return "", storagedriver.ErrUnsupportedMethod{}
}

func (d *driver) getFullPath(subPath string) string {
	return d.mountDir + "/" + subPath
}

//继承自storagedriver的FileWriter
type fileWriter struct {
	vol       *gfapi.Volume
	file      *gfapi.File
	size      int64
	bw        *bufio.Writer
	closed    bool
	committed bool
	cancelled bool
}

func (d *driver) newFileWriter(file *gfapi.File, offset int64) storagedriver.FileWriter {
	return &fileWriter{
		vol:  d.vol,
		file: file,
		size: offset,
		bw:   bufio.NewWriter(file),
	}
}

func (fw *fileWriter) Write(p []byte) (int, error) {
	if fw.closed {
		return 0, fmt.Errorf("already closed")
	} else if fw.committed {
		return 0, fmt.Errorf("already committed")
	} else if fw.cancelled {
		return 0, fmt.Errorf("already cancelled")
	}
	n, err := fw.bw.Write(p)
	fw.size += int64(n)
	return n, err
}

func (fw *fileWriter) Size() int64 {
	return fw.size
}

func (fw *fileWriter) Close() error {
	if fw.closed {
		return fmt.Errorf("already closed")
	}

	if err := fw.bw.Flush(); err != nil {
		return err
	}

	if err := fw.file.Sync(); err != nil {
		return err
	}

	if err := fw.file.Close(); err != nil {
		return err
	}
	fw.closed = true
	return nil
}

func (fw *fileWriter) Cancel() error {
	if fw.closed {
		return fmt.Errorf("already closed")
	}

	fw.cancelled = true
	fw.file.Close()
	return fw.vol.Rmdir(fw.file.Name()) //这句感觉可能有问题，volume里面找不着删除文件的方法
}

func (fw *fileWriter) Commit() error {
	if fw.closed {
		return fmt.Errorf("already closed")
	} else if fw.committed {
		return fmt.Errorf("already committed")
	} else if fw.cancelled {
		return fmt.Errorf("already cancelled")
	}

	if err := fw.bw.Flush(); err != nil {
		return err
	}

	if err := fw.file.Sync(); err != nil {
		return err
	}

	fw.committed = true
	return nil
}

type fileInfo struct {
	os.FileInfo
	path string
}

var _ storagedriver.FileInfo = fileInfo{}

// Path provides the full path of the target of this file info.
func (fi fileInfo) Path() string {
	return fi.path
}

// Size returns current length in bytes of the file. The return value can
// be used to write to the end of the file at path. The value is
// meaningless if IsDir returns true.
func (fi fileInfo) Size() int64 {
	if fi.IsDir() {
		return 0
	}

	return fi.FileInfo.Size()
}

// ModTime returns the modification time for the file. For backends that
// don't have a modification time, the creation time should be returned.
func (fi fileInfo) ModTime() time.Time {
	return fi.FileInfo.ModTime()
}

// IsDir returns true if the path is a directory.
func (fi fileInfo) IsDir() bool {
	return fi.FileInfo.IsDir()
}
