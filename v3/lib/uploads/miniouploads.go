package uploads

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"regexp"
	"strings"

	wst "github.com/fredyk/westack-go/v3/common"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioUpload struct {
	File      multipart.FileHeader `json:"file"`
	Name      string               `json:"name"`
	Directory string               `json:"directory"`
}

type MinioDownload struct {
	FileName  string `json:"fileName"`
	Directory string `json:"directory"`
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
	endpoint := client.Endpoint
	accessKeyID := client.AccessKey
	secretAccessKey := client.SecretKey
	// Initialize minio client object.
	minioClient, errInit := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: true,
	})
	if errInit != nil {
		return nil, errInit
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
		fmt.Printf("Error opening file: %v\n", err)
		return minioUploadResponse, err
	}
	defer buffer.Close()

	// Create minio connection.
	minioClient, err := minioConnection(client)
	if err != nil {
		fmt.Printf("Error creating minio connection: %v\n", err)
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
	contentType := wst.CleanContentType(file.Header.Get("Content-Type"))
	fileSize := file.Size

	// Upload the zip File with PutObject
	info, err := minioClient.PutObject(ctx, bucketName, objectName, fileBuffer, fileSize, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		fmt.Printf("Error uploading file: %v\n", err)
		return minioUploadResponse, err
	}
	resInfo := fmt.Sprintf("%s/%s/%s", client.PublicUrl, info.Bucket, info.Key)

	minioUploadResponse.Url = resInfo

	return minioUploadResponse, nil
}

func (client MinioClient) DownloadFile(download MinioDownload) (io.Reader, error) {
	ctx := context.Background()
	bucketName := client.Bucket
	fileName := download.FileName
	directory := download.Directory

	// Create minio connection.
	minioClient, err := minioConnection(client)
	if err != nil {
		fmt.Printf("Error creating minio connection: %v\n", err)
		return nil, err
	}

	// remove leading and trailing slashes from directory and filename
	directory = strings.Trim(directory, "/")
	fileName = strings.Trim(fileName, "/")

	objectName := directory + "/" + fileName

	// Get the object
	object, err := minioClient.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		fmt.Printf("Error getting object: %v\n", err)
		return nil, err
	}

	return object, nil
}

func CreateRawMultipart(fileName string, directory string, f *os.File, mimeType string) ([]byte, string, error) {

	w := new(strings.Builder)
	writer := multipart.NewWriter(w)
	var boundary string
	part, err := writer.CreatePart(mimeHeader("form-data; name=\"file\"; filename=\""+fileName+"\"", mimeType))
	if err != nil {
		return nil, boundary, err
	}

	b, err := readFile(f)
	if err != nil {
		return nil, boundary, err
	}
	n, err := part.Write(b)
	if err != nil {
		return nil, boundary, err
	}
	fmt.Printf("Wrote %d bytes\n", n)

	_ = writer.WriteField("name", fileName)
	_ = writer.WriteField("directory", directory)

	err = writer.Close()
	if err != nil {
		return nil, boundary, err
	}
	boundary = writer.Boundary()

	//// Now read the part
	//reader := multipart.NewReader(strings.NewReader(writer.FormDataContentType()), boundary)
	//var buff []byte
	//for {
	//	part, err := reader.NextPart()
	//	if err == io.EOF {
	//		break
	//	}
	//	if err != nil {
	//		return nil, boundary, err
	//	}
	//	b, err := io.ReadAll(part)
	//	if err != nil {
	//		return nil, boundary, err
	//	}
	//	buff = append(buff, b...)
	//}

	buff := []byte(w.String())

	return buff, boundary, nil
}

func mimeHeader(head string, mimeType string) textproto.MIMEHeader {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", head)
	header.Set("Content-Type", mimeType)
	return header
}

func readFile(file multipart.File) ([]byte, error) {
	var fileBytes []byte
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return fileBytes, nil
}
