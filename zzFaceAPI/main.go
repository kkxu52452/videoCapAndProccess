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
}

type MyResponse struct {
	Faces 	[4]Location 	`json:"faces"`
}

const zzFace_URL = "https://gateway.qzcloud.com/api/v1/web/zzwu0/face/mtcnnfd.json"

func main() {

	// parse args
	deviceID := os.Args[1]

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

	//color for the rect when faces detected
	blue := color.RGBA{0, 0, 255, 0}

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

		if img.Empty() {
			continue
		}

		imgCopy := img.Clone()
		// for local output
		picName := fmt.Sprintf("%d.jpg", i)

		// encode the img as a JPG image
		imgBytes, _ := gocv.IMEncode(".jpg", imgCopy)

		// convert image to base64 to send it with a json object
		// Thanks to Billzong, without his help I couldn't solve this problem.
		imgBase64 := url.QueryEscape(base64.StdEncoding.EncodeToString(imgBytes))

		// detect faces and measure the time of API call
		start := time.Now()
		resp := callFaceDetecAPI(imgBase64)

		//fmt.Printf("Face Detect Result#%d: %s\n", i, resp.ReturnMsg)
		elapsed := time.Since(start)
		imgText := fmt.Sprintf("Time Consumed: %s, Current Time: %s", elapsed, time.Now().UTC())

		// if there is no face in the img
		if len(resp.Faces) == 0 {
			gocv.IMWrite(picName, imgCopy)
			continue
		}
		// otherwise, draw a rectangle around each face on the image
		for _, f := range resp.Faces {
			gocv.Rectangle(&imgCopy, image.Rect(int(f.Left),int(f.Top),int(f.Width+f.Left),int(f.Height+f.Top)), blue,3)
		}
		gocv.PutText(&imgCopy, imgText, image.Point{50, 50}, gocv.FontHersheyPlain, 1.8, blue, 2)
		gocv.IMWrite(picName, imgCopy)
		//writer.Write(img)
	}
}

func callFaceDetecAPI(imgBase64 string) MyResponse {

	// request payload
	payload := strings.NewReader("image_type=BASE64&image=" + imgBase64)

	req, _ := http.NewRequest("POST", zzFace_URL, payload)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("x-fdn-sign", "k,1560239155,13ef948afdda0efd4fc04416267b458f")

	// check if new client is created every call, package FIN
	// https://medium.com/@nate510/don-t-use-go-s-default-http-client-4804cb19f779
	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()

	var resp MyResponse
	err := json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		panic(err)
	}

	return resp
}
