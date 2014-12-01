package adaptor

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/compose/transporter/pkg/message"
	"github.com/compose/transporter/pkg/pipe"
)

type File struct {
	uri  string
	pipe *pipe.Pipe

	filehandle *os.File
}

func NewFile(p *pipe.Pipe, extra Config) (StopStartListener, error) {

	var (
		conf FileConfig
		err  error
	)
	if err = extra.Construct(&conf); err != nil {
		return nil, err
	}

	return &File{
		uri:  conf.Uri,
		pipe: p,
	}, nil
}

/*
 * start the module
 * TODO: we only know how to listen on stdout for now
 */

func (d *File) Start() (err error) {
	defer func() {
		d.Stop()
	}()

	return d.readFile()
}

func (d *File) Listen() (err error) {
	defer func() {
		d.Stop()
	}()

	if strings.HasPrefix(d.uri, "file://") {
		filename := strings.Replace(d.uri, "file://", "", 1)
		d.filehandle, err = os.Create(filename)
		if err != nil {
			d.pipe.Err <- err
			return err
		}
	}

	return d.pipe.Listen(d.dumpMessage)
}

/*
 * stop the capsule
 */
func (d *File) Stop() error {
	d.pipe.Stop()
	return nil
}

/*
 * read each message from the file
 */
func (d *File) readFile() (err error) {
	filename := strings.Replace(d.uri, "file://", "", 1)
	d.filehandle, err = os.Open(filename)
	if err != nil {
		d.pipe.Err <- err
		return err
	}

	decoder := json.NewDecoder(d.filehandle)
	for {
		var doc map[string]interface{}
		if err := decoder.Decode(&doc); err == io.EOF {
			break
		} else if err != nil {
			d.pipe.Err <- err
			return err
		}
		d.pipe.Send(message.NewMsg(message.Insert, d.uri, doc))
	}
	return nil
}

/*
 * dump each message to the file
 */
func (d *File) dumpMessage(msg *message.Msg) (*message.Msg, error) {
	jdoc, err := json.Marshal(msg.Document())
	if err != nil {
		return msg, fmt.Errorf("can't unmarshal doc %v", err)
	}

	if strings.HasPrefix(d.uri, "stdout://") {
		fmt.Println(string(jdoc))
	} else {
		_, err = fmt.Fprintln(d.filehandle, string(jdoc))
		if err != nil {
			return msg, err
		}
	}

	return msg, nil
}

type FileConfig struct {
	Uri string `json:"uri"`
}