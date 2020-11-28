package main

import (
	"io/ioutil"
	"log"
	"os"
	"sync"
	"testing"
)

func TestResizeInLocal(t *testing.T) {
	// For ease of use, just put images in the same directory as the program
	//imageToTest := [...]string{"muse-brick-art-photo.jpeg"}
	argsWithoutProg := os.Args[2:] // ignore program name and test thing - also, assuming well intentioned input
	log.Printf("args: %v", argsWithoutProg)
	for _, photoPath := range argsWithoutProg {
		photoBytes, err := ioutil.ReadFile(photoPath)
		if err != nil {
			log.Printf("Error in opening file: %v", err)
			return
		}

		fileWithoutExtension, fileExt := getImageNameAndExt(photoPath)
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
				if err := ioutil.WriteFile(fileWithoutExtension+size.Suffix+fileExt, resizedImage, 0666); err != nil {
					log.Printf("error writing %s", fileWithoutExtension+size.Suffix+fileExt)
					return
				}
			}()
		}
		wg.Wait()
		log.Printf("Resized test image %s", photoPath)
	}
}
