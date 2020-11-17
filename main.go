package main

import (
	//"context"
	"bytes"
	"image/jpeg"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/disintegration/imaging"
)

type PhotoSize struct {
	Width  int
	Suffix string
}

var defaultPhotoSizes = []PhotoSize{
	PhotoSize{
		Width:  333,
		Suffix: ".jpg",
	},
	PhotoSize{
		Width:  667,
		Suffix: "@2x.jpg",
	},
	PhotoSize{
		Width:  1500,
		Suffix: "_large.jpg",
	},
	PhotoSize{
		Width:  3000,
		Suffix: "_large@2x.jpg",
	},
}

//This lambda will only be called on CREATE object events

func HandleRequest(_, event events.S3Event) (string, error) {

	// Even if multiple events are sent, should be from same src
	// Maybe later spin off a concurrent function for every record
	for _, record := range event.Records {
		s3Ses, err := session.NewSession(&aws.Config{
			Region: &record.AWSRegion,
		})

		if err != nil {
			log.Printf("Could not instantiate s3 session")
		}

		// Should I add the bucket names as env vars
		// or define them here somehow
		srcBucket := record.S3.Bucket.Name
		srcKey := record.S3.Object.Key

		photoBuff := aws.WriteAtBuffer{} // Store the picture to memory
		s3downloader := s3manager.NewDownloader(s3Ses)
		_, err = s3downloader.Download(&photoBuff, &s3.GetObjectInput{
			Bucket: &srcBucket,
			Key:    &srcKey,
		})

		if err != nil {
			log.Printf("Could not download %s from bucket %s", srcKey, srcBucket)
		}

		imageBytes := photoBuff.Bytes()
		reader := bytes.NewReader(imageBytes)
		img, err := jpeg.Decode(reader)

		if err != nil {
			log.Printf("Could not decode downloaded image %s", srcKey)
		}

		fileExt := filepath.Ext(srcKey)
		fileWithoutExtension := strings.TrimSuffix(srcKey, fileExt)
		s3uploader := s3manager.NewUploader(s3Ses)

		for _, size := range defaultPhotoSizes {
			log.Printf("Image: %s - Creating size: %v\n", srcKey, size)
			newImage := imaging.Resize(img, size.Width, 0, imaging.Lanczos)
			newName := fileWithoutExtension + size.Suffix
			newBuf := new(bytes.Buffer)
			err := jpeg.Encode(newBuf, newImage, nil)

			if err != nil {
				log.Printf("Error encoding image to buffer: %v\n", err)
			}

			res, err := s3uploader.Upload(&s3manager.UploadInput{
				Body:   bytes.NewReader(newBuf.Bytes()),
				Bucket: aws.String(os.Getenv("RESIZED_PHOTOS_BUCKET")),
				Key:    &newName,
			})

			if err != nil {
				log.Printf("Failed to upload %s\n", newName)
			}

			log.Printf("Uploaded %s successfully to %s\n", newName, res.Location)
		}
	}
	return "Succesfully processed req", nil
}

// This works and does what I want so we're mostly good
//func testThis() {
//src, err := imaging.Open("gardens_by_the_bay_brandur.jpg")
//if err != nil {
//log.Fatalf("failed to open image: %v\n", err)
//}

//for _, size := range defaultPhotoSizes {
//fmt.Sprint("Creating size: %v\n", size)
//newImage := imaging.Resize(src, size.Width, 0, imaging.Lanczos) // height as 0 preserves aspect ratio
//newPath := "gardens_by_the_bay_brandur" + size.Suffix
//err = imaging.Save(newImage, newPath)
//if err != nil {
//log.Fatalf("Failed to save image %v", err)
//}
//}
//}

func main() {
	lambda.Start(HandleRequest)
}
