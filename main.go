package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

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
	ReturnMsg string 	`json:"error_msg"`
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

	//if ok := webcam.Read(&img); !ok {
	//	fmt.Printf("Cannot read device %v\n", deviceID)
	//	return
	//}

	// color for the rect when faces detected
	//blue := color.RGBA{0, 0, 255, 0}

	//writer, err := gocv.VideoWriterFile(saveFile, "MJPG", 25, img.Cols(), img.Rows(), true)
	//if err != nil {
	//	fmt.Printf("error opening video writer device: %v\n", saveFile)
	//	return
	//}
	//defer writer.Close()

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
		fmt.Printf("Face Detect Result#%d: %s", i, resp.ReturnMsg)

		// draw a rectangle around each face on the original image,
		// along with
		details := resp.DetecResult.FaceList
		for _, d := range details {
			fmt.Println(d.Location)
			//rotrect := gocv.RotatedRect{nil,nil,image.Point{loc.Left,loc.Top},loc.width,loc.height,loc.rotation}

			//gocv.Rectangle(&img, image.Rect(loc.Left,loc.Top,loc.width+loc.Left,loc.height+loc.Top), blue, 3)
			//size := gocv.GetTextSize("Human", gocv.FontHersheyPlain, 1.2, 2)
			//pt := image.Pt(r.Min.X+(r.Min.X/2)-(size.X/2), r.Min.Y-2)
			//gocv.PutText(&img, "Human", pt, gocv.FontHersheyPlain, 1.2, blue, 2)
		}

		//writer.Write(img)
	}
}

func callFaceDetecAPI(img gocv.Mat) MyResponse {

	buf, err := gocv.IMEncode(".jpg", img)

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
