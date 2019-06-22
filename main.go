package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"gocv.io/x/gocv"
)

type Location struct {
	Left 		float64
	Top  		float64
	Width 		float64
	Height 		float64
	Rotation 	float64
}

type Detail struct {
	Location 		Location   	`json:"location"`
	Probability 	float64 	`json:"face_probability"`
}

type Result struct {
	FaceList []Detail  `json:"face_list"`
}

type MyResponse struct {
	ReturnMsg 	string 	`json:"error_msg"`
	DetecResult Result  `json:"result"`
}

const URL = "https://aip.baidubce.com/rest/2.0/face/v3/detect?access_token=24.455a0daccbd329c48d63307cfc3ac5f8.2592000.1563431485.282335-16550271"

func main() {

	// parse args
	deviceID := os.Args[1]
	//saveFile := os.Args[2]

	// open webcam
	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		fmt.Printf("error opening video capture device: %v\n", deviceID)
		return
	}
	defer webcam.Close()

	// prepare image matrix
	img := gocv.NewMat()
	defer img.Close()

	// use a mutex to safely access 'img' across multiple goroutines
	var mutex = &sync.Mutex{}

	//if ok := webcam.Read(&img); !ok {
	//	fmt.Printf("Cannot read device %v\n", deviceID)
	//	return
	//}

	//color for the rect when faces detected
	blue := color.RGBA{0, 0, 255, 0}

	//writer, err := gocv.VideoWriterFile(saveFile, "MJPG", 25, img.Cols(), img.Rows(), true)
	//if err != nil {
	//	fmt.Printf("error opening video writer device: %v\n", saveFile)
	//	return
	//}
	//defer writer.Close()

	fmt.Printf("Start reading device: %v\n", deviceID)
	// read frame continuously to keep buffer updated
	go func() {

		var sum 	int64  			//Total time used on reading images from webcam
		var count 	int64
		var start	time.Time
		var elapsed time.Duration
		for {
			start = time.Now()
			mutex.Lock()
			if ok := webcam.Read(&img); !ok {
				fmt.Printf("Device closed: %v\n", deviceID)
				return
			}
			mutex.Unlock()
			elapsed = time.Since(start)

			sum = sum + elapsed.Nanoseconds()
			count++
			//time.Sleep(200 * time.Millisecond)
			//fmt.Printf("[MEASURE]Average time of reading a image: %d ms\n", sum/count/1000000)
		}
	}()

	// make sure that the goroutine executes at least once before going on
	time.Sleep(500 * time.Millisecond)

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
		// detect faces and measure the time of API call
		start := time.Now()
		resp := callFaceDetecAPI(imgCopy)

		//fmt.Printf("Face Detect Result#%d: %s\n", i, resp.ReturnMsg)
		elapsed := time.Since(start)
		imgText := fmt.Sprintf("Result: %s, Time Consumed: %s, Current Time: %s", resp.ReturnMsg, elapsed, time.Now().UTC())

		// if there is no face in the img
		if len(resp.DetecResult.FaceList) == 0 {
			gocv.PutText(&imgCopy, imgText, image.Point{50, 50}, gocv.FontHersheyPlain, 1.8, blue, 2)
			gocv.IMWrite(picName, imgCopy)
			continue
		}
		// otherwise, draw a rectangle around each face on the image
		details := resp.DetecResult.FaceList
		for _, d := range details {
			loc := d.Location
			gocv.PutText(&imgCopy, imgText, image.Point{50, 50}, gocv.FontHersheyPlain, 1.8, blue, 2)
			gocv.Rectangle(&imgCopy, image.Rect(int(loc.Left),int(loc.Top),int(loc.Width+loc.Left),int(loc.Height+loc.Top)), blue, 3)

			//gocv.PutText(&img, "Human", pt, gocv.FontHersheyPlain, 1.2, blue, 2)
		}
		gocv.IMWrite(picName, imgCopy)
		//writer.Write(img)
	}
}

func callFaceDetecAPI(img gocv.Mat) MyResponse {

	// encodes an image Mat into a memory buffer using the image format passed in
	buf, err := gocv.IMEncode(".jpg", img)

	// Thanks to Billzong, without his help I couldn't solve this problem.
	imgBase64 := url.QueryEscape(base64.StdEncoding.EncodeToString(buf))

	payload := strings.NewReader("image_type=BASE64&image=" + imgBase64)

	req, _ := http.NewRequest("POST", URL, payload)

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept-Type", "application/json")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()

	var resp MyResponse
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		panic(err)
	}

	return resp
}
