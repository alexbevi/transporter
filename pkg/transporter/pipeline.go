package transporter

/*
 * A Pipeline is a the end to end description of a transporter data flow.
 * including the source, sink, and all the transformers along the way
 */

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/compose/transporter/pkg/pipe"
)

const (
	VERSION = "0.0.1"
)

type Pipeline struct {
	api Api

	source *Node
	// chunks []pipelineChunk

	// nodeWg    *sync.WaitGroup
	metricsWg *sync.WaitGroup
}

// NewPipeline creates a new Transporter Pipeline, with the given node acting as the Source.
// subsequent nodes should be added via AddNode
func NewPipeline(source *Node, api Api) (*Pipeline, error) {
	pipeline := &Pipeline{
		api: api,
		// chunks:    make([]pipelineChunk, 0),
		// nodeWg:    &sync.WaitGroup{},
		metricsWg: &sync.WaitGroup{},
	}

	fmt.Printf("api: %+v\n\n", api)
	source.DoTheThingWeNeedToDo(api)

	// sourcePipe := pipe.NewSourcePipe(source.Name, time.Duration(api.MetricsInterval)*time.Millisecond)
	// err := source.actualize(sourcePipe)
	// if err != nil {
	// 	return pipeline, err
	// }

	fmt.Println("2")

	pipeline.source = source //pipelineSource{config: source, node: node, pipe: sourcePipe}

	fmt.Println("3")

	go pipeline.startErrorListener(source.pipe.Err)
	go pipeline.startEventListener(source.pipe.Event)

	return pipeline, nil
}

// lastPipe returns either the source pipe, or the pipe of the most recently added node.
// we use this to generate a new pipe
// func (pipeline *Pipeline) lastPipe() pipe.Pipe {
// 	if len(pipeline.chunks) == 0 {
// 		return pipeline.source.pipe
// 	}
// 	return pipeline.chunks[len(pipeline.chunks)-1].pipe
// }

// AddNode adds a node to the pipeline
// func (pipeline *Pipeline) AddNode(config ConfigNode) error {
// 	return pipeline.addNode(config, pipe.NewJoinPipe(pipeline.lastPipe(), config.Name))
// }

// AddTerminalNode adds the last node in the pipeline.
// The last node is different only because we use a pipe.SinkPipe instead of a JoinPipe.
// func (pipeline *Pipeline) AddTerminalNode(config ConfigNode) error {
// 	return pipeline.addNode(config, pipe.NewSinkPipe(pipeline.lastPipe(), config.Name))
// }

// addNode creates the node from the ConfigNode and adds it to the list of nodes
// func (pipeline *Pipeline) addNode(config ConfigNode, p pipe.Pipe) error {
// 	node, err := config.Create(p)
// 	if err != nil {
// 		return err
// 	}
// 	n := pipelineChunk{config: config, node: node, pipe: p}
// 	pipeline.chunks = append(pipeline.chunks, n)
// 	return nil
// }

// func (pipeline *Pipeline) String() string {
// 	out := " - Pipeline\n"
// 	out += fmt.Sprintf("  - Source: %s\n", pipeline.source.config)
// 	if len(pipeline.chunks) > 1 {
// 		for _, t := range pipeline.chunks[1 : len(pipeline.chunks)-1] {
// 			out += fmt.Sprintf("   - %s\n", t)
// 		}
// 	}
// 	if len(pipeline.chunks) >= 1 {
// 		out += fmt.Sprintf("  - Sink:   %s\n", pipeline.chunks[len(pipeline.chunks)-1].config)
// 	}
// 	return out
// }

// Stop sends a stop signal to all the nodes, whether they are running or not
func (pipeline *Pipeline) Stop() {
	pipeline.source.Stop()
	// for _, chunk := range pipeline.chunks {
	// 	chunk.node.Stop()
	// }
}

// run the pipeline
func (pipeline *Pipeline) Run() error {
	// for _, chunk := range pipeline.chunks {
	// 	go func(node NodeImpl) {
	// 		pipeline.nodeWg.Add(1)
	// 		node.Listen()
	// 		pipeline.nodeWg.Done()
	// 	}(chunk.node)
	// }

	fmt.Println("send boot")

	// send a boot event
	pipeline.source.pipe.Event <- pipe.NewBootEvent(time.Now().Unix(), VERSION, pipeline.endpointMap())

	fmt.Println("after sending boot")

	// start the source
	err := pipeline.source.Start()

	// the source has exited, stop all the other nodes
	pipeline.Stop()

	// pipeline.nodeWg.Wait()
	pipeline.metricsWg.Wait()

	// send a boot event
	pipeline.source.pipe.Event <- pipe.NewExitEvent(time.Now().Unix(), VERSION, pipeline.endpointMap())

	return err
}

func (pipeline *Pipeline) endpointMap() map[string]string {
	return pipeline.source.Endpoints()
	// m := make(map[string]string)
	// m[pipeline.source.Name] = pipeline.source.Type
	// for _, v := range pipeline.chunks {
	// 	m[v.config.Name] = v.config.Type
	// }
	// return m
}

// start error listener consumes all the events on the pipe's Err channel, and stops the pipeline
// when it receives one
func (pipeline *Pipeline) startErrorListener(cherr chan error) {
	for err := range cherr {
		fmt.Printf("Pipeline error %v\nShutting down pipeline\n", err)
		pipeline.Stop()
	}
}

// startEventListener consumes all the events from the pipe's Event channel, and posts them to the ap
func (pipeline *Pipeline) startEventListener(chevent chan pipe.Event) {
	for event := range chevent {
		ba, err := json.Marshal(event)
		if err != err {
			pipeline.source.pipe.Err <- err
			continue
		}
		pipeline.metricsWg.Add(1)
		go func() {
			defer pipeline.metricsWg.Done()
			if pipeline.api.Uri != "" {
				req, err := http.NewRequest("POST", pipeline.api.Uri, bytes.NewBuffer(ba))
				req.Header.Set("Content-Type", "application/json")
				if len(pipeline.api.Pid) > 0 && len(pipeline.api.Key) > 0 {
					req.SetBasicAuth(pipeline.api.Pid, pipeline.api.Key)
				}
				cli := &http.Client{}
				resp, err := cli.Do(req)
				if err != nil {
					fmt.Println("event send failed")
					pipeline.source.pipe.Err <- err
					return
				}

				defer resp.Body.Close()
				body, err := ioutil.ReadAll(resp.Body)

				if resp.StatusCode != 200 && resp.StatusCode != 201 {
					pipeline.source.pipe.Err <- fmt.Errorf("Event Error: http error code, expected 200 or 201, got %d.  %d\n\t%s", resp.StatusCode, resp.StatusCode, body)
					return
				}
				resp.Body.Close()
			}
		}()
		if pipeline.api.Uri != "" {
			fmt.Printf("sent pipeline event: %s -> %s\n", pipeline.api.Uri, event)
		}

	}
}

// pipelineChunk keeps a copy of the config beside the actual node implementation,
// so that we don't have to force fit the properties of the config
// into nodes that don't / shouldn't care about them.
// type pipelineChunk struct {
// 	config ConfigNode
// 	node   NodeImpl
// 	pipe   pipe.Pipe
// }

// pipelineSource is the source node, pipeline and config
// type pipelineSource struct {
// 	config ConfigNode
// 	node   SourceImpl
// 	pipe   pipe.Pipe
// }

/*
n = NewNode(...)

transformer := NewNode(....)
Sink1 := NewNode(....)
Sink2 ;= NewNode(...)
Sink3 := NewNode(...)

transformer.attach(Sink1)
transformer.attach(Sink2)
transformer.attach(transformer2)

n.Attach(transformer)
n.Attach(Sink3)






pipeline := NewPipeline(n)
pipeline.Run

----

source = Source(....)

source.save(sink1 hash)



transformer = Transformer(.....)
transformer.save(....)
transformer.save(....)

source.
 source.transform(transformer)


*/
