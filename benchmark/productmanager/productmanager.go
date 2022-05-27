package productmanager

import (
	"encoding/json"
	"log"
	"os"
)

const (
	IMAGEHASHES_DATA_FILENAME = "image_hashes.json"
)

var (
	imageHashes map[string]string
)

type ImageHash struct {
	Name string
	Hash string
}

type ImageHashesBlob struct {
	ImageHashes []ImageHash `json:"image_hashes"`
}

func init() {
	initImageHashes(IMAGEHASHES_DATA_FILENAME)
}

func initImageHashes(filename string) {
	jsonFromFile, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	var jsonData ImageHashesBlob
	err = json.Unmarshal(jsonFromFile, &jsonData)
	if err != nil {
		log.Fatal(err)
	}

	imageHashes = make(map[string]string)
	for _, imageHash := range jsonData.ImageHashes {
		imageHashes[imageHash.Name] = imageHash.Hash
	}
}

func GetNumOfProducts() int {
	return len(imageHashes)
}

func GetImageHash(key string) string {
	return imageHashes[key]
}
