package main

import (
	"C"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

//export Proc
func Proc(encoded_json *C.char) *C.char {
	message := C.GoString(encoded_json)

	var data []map[string]any
	json.Unmarshal([]byte(message), &data)

	wg := sync.WaitGroup{}
	mux := sync.Mutex{}
	results := make([]map[string]any, 0, len(data))
	for _, item := range data {
		wg.Add(1)
		go func(item map[string]any) {
			defer wg.Done()
			mux.Lock()
			request(fmt.Sprintf("Processing item: %v\n", item))
			results = append(results, item)
			mux.Unlock()
		}(item)
	}
	wg.Wait()

	result, err := json.Marshal(results)
	if err != nil {
		return C.CString(fmt.Sprintf("Error: %v", err))
	}

	return C.CString(string(result))
}

func request(msg string) string {
	fmt.Print(msg)
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	return msg
}

func main() {}
