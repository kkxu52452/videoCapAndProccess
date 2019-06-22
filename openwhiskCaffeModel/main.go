// How to run:
//
// 		go run ./cmd/dnn-detection/main.go [videosource] [modelfile] [configfile] ([backend] [device])
//

package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gocv.io/x/gocv"
)

func main() {

	// parse args
	deviceID := os.Args[1]
	model := os.Args[2]
	config := os.Args[3]
	backend := gocv.NetBackendDefault
	if len(os.Args) > 4 {
		backend = gocv.ParseNetBackend(os.Args[4])
	}

	target := gocv.NetTargetCPU
	if len(os.Args) > 5 {
		target = gocv.ParseNetTarget(os.Args[5])
	}

	// open capture device
	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		fmt.Printf("Error opening video capture device: %v\n", deviceID)
		return
	}
	defer webcam.Close()

	//window := gocv.NewWindow("DNN Detection")
	//defer window.Close()

	img := gocv.NewMat()
	defer img.Close()

	//color for the rect when faces detected
	blue := color.RGBA{0, 0, 255, 0}

	// use a mutex to safely access 'img' across multiple goroutines
	var mutex = &sync.Mutex{}

	fmt.Printf("Start reading device: %v\n", deviceID)
	// read frame continuously to keep buffer updated
	go func() {
		for {
			mutex.Lock()
			if ok := webcam.Read(&img); !ok {
				fmt.Printf("Device closed: %v\n", deviceID)
				return
			}
			mutex.Unlock()
		}
	}()

	// open DNN object tracking model
	net := gocv.ReadNet(model, config)
	if net.Empty() {
		fmt.Printf("Error reading network model from : %v %v\n", model, config)
		return
	}
	defer net.Close()
	net.SetPreferableBackend(gocv.NetBackendType(backend))
	net.SetPreferableTarget(gocv.NetTargetType(target))

	var ratio float64
	var mean gocv.Scalar
	var swapRGB bool

	if filepath.Ext(model) == ".caffemodel" {
		ratio = 1.0
		mean = gocv.NewScalar(104, 177, 123, 0)
		swapRGB = false
	} else {
		ratio = 1.0 / 127.5
		mean = gocv.NewScalar(127.5, 127.5, 127.5, 0)
		swapRGB = true
	}

	for i := 0; i < 50; i++ {
		//if ok := webcam.Read(&img); !ok {
		//	fmt.Printf("Device closed: %v\n", deviceID)
		//	return
		//}
		if img.Empty() {
			continue
		}

		imgCopy := img.Clone()
		// for output
		picName := fmt.Sprintf("%d.jpg", i)

		// detect faces and measure the time of model inference
		start := time.Now()
		//fmt.Printf("Face Detect Result#%d: %s\n", i, resp.ReturnMsg)

		// convert image Mat to 300x300 blob that the object detector can analyze
		blob := gocv.BlobFromImage(imgCopy, ratio, image.Pt(300, 300), mean, swapRGB, false)

		// feed the blob into the detector
		net.SetInput(blob, "")

		// run a forward pass thru the network
		prob := net.Forward("")

		if prob.Total() == 0 {
			elapsed := time.Since(start)
			imgText := fmt.Sprintf("Found no face in the Image; Time Consumed: %s; Current Time: %s", elapsed, time.Now().UTC())
			gocv.PutText(&imgCopy, imgText, image.Point{50, 50}, gocv.FontHersheyPlain, 1.8, blue, 2)
			gocv.IMWrite(picName, imgCopy)
			continue
		}

		performDetection(&imgCopy, prob)
		elapsed := time.Since(start)
		imgText := fmt.Sprintf("Found %d face in the Image; Time Consumed: %s; Current Time: %s", prob.Total(), elapsed, time.Now().UTC())
		gocv.PutText(&imgCopy, imgText, image.Point{50, 50}, gocv.FontHersheyPlain, 1.8, blue, 2)
		gocv.IMWrite(picName, imgCopy)

		prob.Close()
		blob.Close()

		//window.IMShow(img)
		//if window.WaitKey(1) >= 0 {
		//	break
		//}
	}
}

// performDetection analyzes the results from the detector network,
// which produces an output blob with a shape 1x1xNx7
// where N is the number of detections, and each detection
// is a vector of float values
// [batchId, classId, confidence, left, top, right, bottom]
func performDetection(frame *gocv.Mat, results gocv.Mat) {
	for i := 0; i < results.Total(); i += 7 {
		confidence := results.GetFloatAt(0, i+2)
		if confidence > 0.5 {
			left := int(results.GetFloatAt(0, i+3) * float32(frame.Cols()))
			top := int(results.GetFloatAt(0, i+4) * float32(frame.Rows()))
			right := int(results.GetFloatAt(0, i+5) * float32(frame.Cols()))
			bottom := int(results.GetFloatAt(0, i+6) * float32(frame.Rows()))
			gocv.Rectangle(frame, image.Rect(left, top, right, bottom), color.RGBA{0, 255, 0, 0}, 2)
		}
	}
}

