package node

/*
 * A Pipeline is a the end to end description of a transporter data flow.
 * including the source, sink, and all the transformers along the way
 */

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/robertkrimen/otto"
)

type Pipeline struct {
	Source       *Node          `json:"source"`
	Sink         *Node          `json:"sink"`
	Transformers []*Transformer `json:"transformers"`
	errChans     []chan error
}

func NewPipeline(source *Node) *Pipeline {
	return &Pipeline{Source: source, Transformers: make([]*Transformer, 0)}
}

/*
 * create a new pipeline from a value, such as what we would get back
 * from an otto.Value.  basically a pipeline that has lost it's identify,
 * and been interfaced{}
 */
func InterfaceToPipeline(val interface{}) (Pipeline, error) {
	t := Pipeline{}
	ba, err := json.Marshal(val)

	if err != nil {
		return t, err
	}

	err = json.Unmarshal(ba, &t)
	return t, err
}

/*
 * turn this pipeline into an otto Object
 */
func (t *Pipeline) Object() (*otto.Object, error) {
	vm := otto.New()
	ba, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	return vm.Object(fmt.Sprintf(`(%s)`, string(ba)))
}

/*
 * add a transformer function to a pipeline.
 * transformers will be called in fifo order
 */
func (p *Pipeline) AddTransformer(t *Transformer) {
	p.Transformers = append(p.Transformers, t)
}

func (p *Pipeline) String() string {
	out := " - Pipeline\n"
	out += fmt.Sprintf("  - Source: %s\n  - Sink:   %s\n  - Transformers:\n", p.Source, p.Sink)
	for _, t := range p.Transformers {
		out += fmt.Sprintf("   - %s\n", t)
	}
	return out
}

/*
 * Create the pipeline, and instantiate all the nodes
 */
func (p *Pipeline) Create() error {
	err := p.Source.Create(SOURCE)
	if err != nil {
		return err
	}

	err = p.Sink.Create(SINK)
	if err != nil {
		return err
	}

	return nil
}

/*
 * run the pipeline
 */
func (p *Pipeline) Run() error {
	// remember all the errChans
	p.errChans = make([]chan error, len(p.Transformers)+2)

	sourcePipe := NewPipe()
	p.errChans[0] = sourcePipe.Err

	sinkPipe := JoinPipe(sourcePipe)
	p.errChans[1] = sinkPipe.Err

	p.startErrorListener()

	go p.Sink.NodeImpl.Start(sinkPipe)
	return p.Source.NodeImpl.Start(sourcePipe)
}

func (p *Pipeline) startErrorListener() {
	go func(c <-chan time.Time) {
		for _ = range c {
			for i, v := range p.errChans {
				select {
				case err := <-v:
					fmt.Printf("Pipeline node(%d) error %v\n", i, err)
				default:
					//
				}
			}
		}
	}(time.NewTicker(500 * time.Millisecond).C)
}
