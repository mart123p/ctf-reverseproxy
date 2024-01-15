package cbroadcast

import (
	"fmt"
	"sync"
)

// Channel used to send struct
type Channel chan interface{}
type channelStruct struct {
	ready       bool
	subscribers []Channel
	closer      chan bool
	bufferSize  int
}

var mutex sync.Mutex
var channels map[string]*channelStruct
var bufferMaxCallback func(name string)

// Register a new channel with a name to subscribe. The sizeOfBuffer is the size of the queue. Returns a channel that is triggered if the buffer is ever full
func Register(name string, sizeOfBuffer int) {
	mutex.Lock()
	if channels == nil {
		channels = make(map[string]*channelStruct)
	}
	mutex.Unlock()

	channelStruct := channelStruct{
		ready:      true,
		closer:     make(chan bool),
		bufferSize: sizeOfBuffer,
	}
	// Lock the map to write the channel
	mutex.Lock()
	channels[name] = &channelStruct
	mutex.Unlock()
}

// NonBlockingBuffer if the buffer limit is hitted this function is called. Can be used to report if the server is under an excessive load
func NonBlockingBuffer(callback func(name string)) {
	defer mutex.Unlock()
	mutex.Lock()
	bufferMaxCallback = callback
}

// Subscribe to a type of event. Returns a channel needed to communicate and the closer channel to inform the subscribers if the channel is closing
func Subscribe(name string) (Channel, chan bool) {
	defer mutex.Unlock()
	mutex.Lock()

	if channel, ok := channels[name]; ok {
		reader := make(Channel, channel.bufferSize)
		channels[name].subscribers = append(channel.subscribers, reader)
		return reader, channels[name].closer
	}

	panic(fmt.Sprintf("The channel %s was not registered", name))
}

// Broadcast the data to all the listener. Make sure that this function is not broadcasting in the same go subroutine that it is listening to avoid deadlocks.
// If you need this type of architecture make sure that you increase the size of the buffer
func Broadcast(name string, data interface{}) {
	mutex.Lock()

	if channel, ok := channels[name]; ok {
		subscribers := make([]Channel, len(channel.subscribers))
		copy(subscribers, channel.subscribers)
		bufferMax := bufferMaxCallback
		mutex.Unlock() //Early unlock since we have a copy

		for _, sub := range subscribers {
			if bufferMax == nil {
				sub <- data //This will lock if the buffer is full
			} else {
				select {
				case sub <- data:
				default:
					bufferMax(name) //Call the function to inform that some values are thrown out
				}
			}

		}
	} else {
		mutex.Unlock()
		panic(fmt.Sprintf("The channel %s was not registered", name))
	}
}

// Close Send a signal to all the subscribers that the broadcast is completed
func Close(name string) {
	defer mutex.Unlock()
	if channel, ok := channels[name]; ok {
		close(channel.closer)
		return
	}
	panic(fmt.Sprintf("The channel %s was not registered", name))
}
