package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"gocv.io/x/gocv"
)

type Location struct {
	Left 	int 	`json:"left"`
	Top  	int 	`json:"top"`
	width 	int 	`json:"left"`
	height 	int 	`json:"height"`
	rotation float64 	`json:"rotation"`
}

type Detail struct {
	location 		Location   	`json:"location"`
	Probability 	float64 	`json:"face_probability"`
}

type Result struct {
	face_list []Detail  `json:"face_list"`
}

type Response struct {
	ReturnMsg string 	`json:"error_msg"`
	DetecResult Result  `json:"result"`
}


func main() {

	// parse args
	deviceID := os.Args[1]
	saveFile := os.Args[2]

	// open webcam
	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		fmt.Printf("error opening video capture device: %v\n", deviceID)
		return
	}
	defer webcam.Close()

	// open display window
	window := gocv.NewWindow("Face Detect")
	defer window.Close()

	// prepare image matrix
	img := gocv.NewMat()
	defer img.Close()

	// color for the rect when faces detected
	blue := color.RGBA{0, 0, 255, 0}

	writer, err := gocv.VideoWriterFile(saveFile, "MJPG", 25, img.Cols(), img.Rows(), true)
	if err != nil {
		fmt.Printf("error opening video writer device: %v\n", saveFile)
		return
	}
	defer writer.Close()

	fmt.Printf("Start reading device: %v\n", deviceID)
	for i := 0; i < 100; i++ {
		if ok := webcam.Read(&img); !ok {
			fmt.Printf("Device closed: %v\n", deviceID)
			return
		}
		if img.Empty() {
			continue
		}

		// detect faces
		resp := callFaceDetecAPI(img)
		fmt.Printf("Face Detect: %s", resp.ReturnMsg)

		// draw a rectangle around each face on the original image,
		// along with
		details := resp.DetecResult.face_list
		for _, d := range details {
			//loc := d.location
			//rotrect := gocv.RotatedRect{nil,nil,image.Point{loc.Left,loc.Top},loc.width,loc.height,loc.rotation}

			gocv.Rectangle(&img, image.Rect(50,50,100,100), blue, 3)
			//size := gocv.GetTextSize("Human", gocv.FontHersheyPlain, 1.2, 2)
			//pt := image.Pt(r.Min.X+(r.Min.X/2)-(size.X/2), r.Min.Y-2)
			//gocv.PutText(&img, "Human", pt, gocv.FontHersheyPlain, 1.2, blue, 2)
		}

		writer.Write(img)
	}
}

func callFaceDetecAPI(img gocv.Mat) Response {

	url := "https://aip.baidubce.com/rest/2.0/face/v3/detect?access_token=24.455a0daccbd329c48d63307cfc3ac5f8.2592000.1563431485.282335-16550271"

	payload := base64.StdEncoding.EncodeToString(img.ToBytes())

	req, _ := http.NewRequest("POST", url, strings.NewReader(payload))

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()

	// read json http response
	jsonDataFromHttp, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var resp Response
	err = json.Unmarshal(jsonDataFromHttp, resp)
	if err != nil {
		panic(err)
	}

	return resp
}
