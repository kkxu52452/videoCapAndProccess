package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gocv.io/x/gocv"
)

// JSONMap is raw json object
type JSONMap map[string]interface{}

type HTTPResponse struct {
	Headers    JSONMap     	`json:"headers,omitempty" structs:"headers"`
	StatusCode int         	`json:"statusCode,omitempty" structs:"statusCode"`
	Body       []byte 		`json:"body,omitempty" structs:"body"`
}

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
	FaceResult Result   `json:"result"`
}

type FromFDN struct {
	DetecResult  MyResponse  `json:"detec_result"`
}

const FDN_URL = "https://c6d8574c-4545-4891-96e8-93751b4b0fea:y9bRMbeu1NmQCtzmKKOOxxRxLp8mkssYUrLHtFwrcRvlA7FTymfamtZeCKy9ku44@" +
	"us-south.functions.cloud.ibm.com/api/v1/namespaces/ikoosgg%40hotmail.com_dev/actions/test/IBMbaiduAPI"

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
		result := callFaceDetecAPI(imgCopy)
		res := result.DetecResult
		elapsed := time.Since(start)

		imgText := fmt.Sprintf("Status: %s; Time Consumed: %s; Current Time: %s", res.ReturnMsg, elapsed, time.Now().UTC())
		gocv.PutText(&imgCopy, imgText, image.Point{50, 50}, gocv.FontHersheyPlain, 1.8, blue, 2)
		// if there is no face in the img
		if len(res.DetecResult.FaceList) == 0 {
			gocv.IMWrite(picName, imgCopy)
			continue
		}
		// otherwise, draw a rectangle around each face on the image
		details := res.DetecResult.FaceList
		for _, d := range details {
			loc := d.Location
			gocv.Rectangle(&imgCopy, image.Rect(int(loc.Left),int(loc.Top),int(loc.Width+loc.Left),int(loc.Height+loc.Top)), blue, 3)
		}
		// save as local image
		gocv.IMWrite(picName, imgCopy)
	}
}

func callFaceDetecAPI(img gocv.Mat) FromFDN {

	// encodes an image Mat into a memory buffer using the image format pass in
	buf, _ := gocv.IMEncode(".jpg", img)

	// Thanks to Billzong, without his help I couldn't solve this problem.
	imgBase64 := base64.StdEncoding.EncodeToString(buf)

	payload := strings.NewReader("image_type=BASE64&image=" + imgBase64)

	req, _ := http.NewRequest("POST", FDN_URL, payload)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept-Type", "application/json")

	res, _ := http.DefaultClient.Do(req)

	var result FromFDN
	err := json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		panic(err)
	}

	defer res.Body.Close()

	return result
}
