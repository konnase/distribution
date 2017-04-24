package gluster

import (
	//"io/ioutil"
	//"reflect"
	"fmt"
	//"github.com/docker/distribution/context"
	storagedriver "github.com/docker/distribution/registry/storage/driver"
	//"string"
	"strings"
	"testing"
	"time"
	//"github.com/gluster/gogfapi/gfapi"
	//"github.com/docker/distribution/registry/storage/driver/testsuites"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

var glusterDriverConstructor func() (storagedriver.StorageDriver, error)

func TestNew(t *testing.T) {
	params := &Parameters{
		host:           "localhost",
		volname:        "datapoint",
		mountDirectory: "/mnt/gluster",
	}
	fmt.Printf("Test stotageDriver !\n")
	glusterDriverConstructor = func() (storagedriver.StorageDriver, error) {
		return New(*params)
	}
	d, err := glusterDriverConstructor()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	fmt.Println(d.Name())

	//Test writer()
	fmt.Printf("\nTest Writer !\n")
	f, err4 := d.Writer(nil, "/tests/hello.txt", false)
	if err4 != nil {
		fmt.Printf("%v\n", err4)
		//os.Exit(0)
	}
	var tmp string = "2017-04-21T14:16:45Z"
	cont := []byte(tmp)
	n, err := f.Write(cont)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%d bytes writen to p from filewriter!\n", n)
	f.Commit()
	//f.Cancel()
	f.Close()
	time.Sleep(10 * time.Millisecond)

	//Test reader()
	fmt.Printf("\nTest Reader !\n")
	file, err1 := d.Reader(nil, "/tests/hello.txt", 0)
	if err1 != nil {
		fmt.Printf("%v\n", err1)
	}
	var p0 [100]byte
	p := p0[:]
	n, err2 := file.Read(p)
	if err2 != nil {
		fmt.Printf("%v\n", err2)
	}
	tmp1 := byteToString(p)
	fmt.Println(strings.EqualFold(tmp1, tmp))
	times, err := time.Parse(time.RFC3339, tmp1)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("Time is %v\n", times)
	fmt.Printf("%d bytes read!  !%s!\n", n, string(p))
	file.Close() //io.ReadCloser only has two methods ,that is Read() and Close()

	//Test GetContent()  该方法里面的ioutil.ReadAll(rc)执行后不返回值，有待解决
	// fmt.Printf("\nTest GetContent !\n")
	// contents, err := d.GetContent(nil, "/tests/hello.txt")
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// times, err := time.Parse(time.RFC3339, string(contents))
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Printf("Get contents: %s\n", string(contents))
	// fmt.Printf("Time is %v\n", times)

	//Test PutContent()
	// fmt.Printf("\nTest PutContent !\n")
	// con := "test putcontent"
	// if err = d.PutContent(nil, "/job.txt", []byte(con)); err != nil {
	// 	fmt.Println(err)
	// }

	//Test Stat()
	// fmt.Printf("\nTest Stat !\n")
	// state, err3 := d.Stat(nil, "/job.txt")
	// if err3 != nil {
	// 	fmt.Printf("%v\n", err3)
	// }
	// fmt.Printf("%v\n", state.ModTime())

	// //Test List()
	// fmt.Printf("\nTest List !\n")
	// lists, err := d.List(nil, "/")
	// fmt.Println(lists)

	//Test Move()
	// fmt.Printf("\nTest Move !\n")
	// err = d.Move(nil, "/tests/hello.txt", "/htf/hello.txt")
	// if err != nil {
	// 	fmt.Println(err)
	// }

	//Test Delete()
	// fmt.Printf("\nTest Delete !\n")
	// err = d.Delete(nil, "/test.go")
	// if err != nil {
	// 	fmt.Println(err)
	// }
}

func byteToString(p []byte) string {
	for i := 0; i < len(p); i++ {
		if p[i] == 0 {
			return string(p[0:i])
		}
	}
	return string(p)
}
