package main

import (
	//"context"
	"bytes"
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	//"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/h2non/bimg"
)

type PhotoSize struct {
	Width  int
	Suffix string
}

var defaultPhotoSizes = []PhotoSize{
	PhotoSize{
		Width:  333,
		Suffix: "",
	},
	PhotoSize{
		Width:  667,
		Suffix: "@2x",
	},
	PhotoSize{
		Width:  1500,
		Suffix: "_large",
	},
	PhotoSize{
		Width:  3000,
		Suffix: "_large@2x",
	},
}

//This lambda will only be called on CREATE object events

func HandleRequest(ctx context.Context, event events.S3Event) (string, error) {

	// Even if multiple events are sent, should be from same src
	// Maybe later spin off a concurrent function for every record
	for _, record := range event.Records {
		s3Ses, err := session.NewSession(&aws.Config{
			Region: &record.AWSRegion,
		})

		if err != nil {
			log.Printf("Could not instantiate s3 session")
			return "", err
		}

		// Should I add the bucket names as env vars
		// or define them here somehow
		srcBucket := record.S3.Bucket.Name
		srcKey := record.S3.Object.Key
		fileWithoutExtension, fileExt := getImageNameAndExt(srcKey)

		photoBuff := aws.WriteAtBuffer{} // Store the picture to memory
		s3downloader := s3manager.NewDownloader(s3Ses)
		_, err = s3downloader.Download(&photoBuff, &s3.GetObjectInput{
			Bucket: &srcBucket,
			Key:    &srcKey,
		})

		if err != nil {
			log.Printf("Could not download %s from bucket %s", srcKey, srcBucket)
			return "", err
		}

		if err != nil {
			log.Printf("Could not decode downloaded image %s", srcKey)
			return "", err
		}

		photoBytes := photoBuff.Bytes()
		s3uploader := s3manager.NewUploader(s3Ses)
		var wg sync.WaitGroup
		wg.Add(len(defaultPhotoSizes))

		for _, size := range defaultPhotoSizes {
			size := size
			go func() {
				log.Printf("size in go func is %v", size)
				defer wg.Done()
				resizedImage, err := resizeImage(&photoBytes, &size)
				if err != nil {
					return
				}
				encodeImageAndUploadToS3(resizedImage, fileWithoutExtension, fileExt, size.Suffix, s3uploader)
			}()
		}
		wg.Wait()
		log.Printf("Resized all images for %s", srcKey)
	}
	return "Succesfully processed req", nil
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("function %s took %s", name, elapsed)
}

func getImageNameAndExt(imgPath string) (string, string) {
	fileExt := filepath.Ext(imgPath)
	fileWithoutExtension := strings.TrimSuffix(imgPath, fileExt)
	return fileWithoutExtension, fileExt
}

func resizeImage(imgBytes *[]byte, requestedSize *PhotoSize) ([]byte, error) {
	newImg, err := bimg.NewImage(*imgBytes).Resize(requestedSize.Width, 0)

	if err != nil {
		log.Printf("Could not create image from byte[]")
		return nil, err
	}
	return newImg, nil
}

func encodeImageAndUploadToS3(img []byte, imgName string, imgExt string, newSizeSuffix string, s3uploader *s3manager.Uploader) {
	s3Key := imgName + newSizeSuffix + imgExt

	res, err := s3uploader.Upload(&s3manager.UploadInput{
		Body:        bytes.NewReader(img),
		Bucket:      aws.String(os.Getenv("RESIZED_PHOTOS_BUCKET")),
		Key:         &s3Key,
		ContentType: aws.String("image/jpeg"),
	})

	if err != nil {
		log.Printf("Failed to upload %s\n", s3Key)
		return
	}

	log.Printf("Uploaded %s successfully to %s\n", s3Key, res.Location)
}

func testLocaly() {
	// Open a file in this directory
	defer timeTrack(time.Now(), "testLocally")
	buffer, err := bimg.Read("./snow_mountain_brandur.jpg")
	if err != nil {
		log.Printf("%v", err)
	}

	var wg sync.WaitGroup
	wg.Add(len(defaultPhotoSizes))

	for _, size := range defaultPhotoSizes {
		size := size
		go func() {
			//defer timeTrack(time.Now(), "go("+size.Suffix+")")
			log.Printf("size in go func is %v", size)
			defer wg.Done()
			resizedImage, err := resizeImage(&buffer, &size)
			if err != nil {
				return
			}
			bimg.Write("snow_mountain_brandur"+size.Suffix+".jpg", resizedImage)
		}()
	}
	wg.Wait()
	log.Printf("Resized all images for snow")
}

func main() {
	//lambda.Start(HandleRequest)
	testLocaly()
}
