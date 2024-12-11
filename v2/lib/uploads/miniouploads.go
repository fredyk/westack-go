package uploads

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"mime/multipart"
	"regexp"
	"strings"
)

type MinioUpload struct {
	File      multipart.FileHeader `json:"file"`
	Name      string               `json:"name"`
	Directory string               `json:"directory"`
}

type MinioUploadResponse struct {
	Url string `json:"url"`
}

type MinioClient struct {
	Bucket    string
	Endpoint  string
	AccessKey string
	SecretKey string
	PublicUrl string
	Region    string
}

var nonAlphanumericRegex = regexp.MustCompile(`[^\p{L}\p{N} .]+`)

// minioConnection func for opening minio connection.
func minioConnection(client MinioClient) (*minio.Client, error) {
	ctx := context.Background()
	endpoint := client.Endpoint
	accessKeyID := client.AccessKey
	secretAccessKey := client.SecretKey
	// Initialize minio client object.
	minioClient, errInit := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: true,
	})
	if errInit != nil {
		return minioClient, errInit
	}

	// Make a new Bucket.
	bucketName := client.Bucket

	// Check to see if bucket already exists.
	exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
	if errBucketExists != nil {
		return minioClient, errBucketExists
	} else if !exists {
		return minioClient, fmt.Errorf("bucket %s does not exist", bucketName)
	}

	return minioClient, errInit
}

func (client MinioClient) UploadFile(upload MinioUpload) (MinioUploadResponse, error) {
	ctx := context.Background()
	bucketName := client.Bucket

	file := upload.File
	name := upload.Name

	// Get Buffer from File
	buffer, err := file.Open()
	minioUploadResponse := MinioUploadResponse{}
	if err != nil {
		return minioUploadResponse, err
	}
	defer buffer.Close()

	// Create minio connection.
	minioClient, err := minioConnection(client)
	if err != nil {
		return minioUploadResponse, err
	}

	var fileName string
	if name != "" {
		fileName = name
	} else {
		fileName = file.Filename
	}

	//remove non-alphanumeric characters from filename
	fileName = strings.ToLower(nonAlphanumericRegex.ReplaceAllString(fileName, "-"))
	fileName = strings.TrimSpace(fileName)

	// remove leading and trailing slashes from directory and filename
	upload.Directory = strings.Trim(upload.Directory, "/")
	fileName = strings.Trim(fileName, "/")

	// add the directory
	objectName := upload.Directory + "/" + fileName
	fileBuffer := buffer
	contentType := file.Header.Get("Content-Type")
	fileSize := file.Size

	// Upload the zip File with PutObject
	info, err := minioClient.PutObject(ctx, bucketName, objectName, fileBuffer, fileSize, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return minioUploadResponse, err
	}
	resInfo := fmt.Sprintf("%s/%s/%s", client.PublicUrl, info.Bucket, info.Key)

	minioUploadResponse.Url = resInfo

	return minioUploadResponse, nil
}
