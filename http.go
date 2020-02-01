package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

//HTTPGateway struct
type HTTPGateway struct {
	Client *http.Client
}

//PostWithApplicationForm func
func (hg *HTTPGateway) PostWithApplicationForm(resourceURL string, parameters map[string]interface{}) (int, error) {
	//prepare parameters
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	for key, value := range parameters {
		if file, ok := value.(*os.File); ok {
			if err := appendFile(key, file, writer); err != nil {
				return 0, errors.WithStack(err)
			}
		}
		if str, ok := value.(string); ok {
			writeFieldErr := writer.WriteField(key, str)
			if writeFieldErr != nil {
				return 0, errors.WithStack(writeFieldErr)
			}
		}
	}
	writeErr := writer.Close()
	if writeErr != nil {
		return 0, errors.WithStack(writeErr)
	}
	//do request
	return hg.doRequestWithContentType(resourceURL, "application/x-www-form-urlencoded", body)
}

func appendFile(parameterName string, file *os.File, requestWriter *multipart.Writer) error {
	part, err := requestWriter.CreateFormFile(parameterName, filepath.Base(file.Name()))
	if err != nil {
		return errors.WithStack(err)
	}
	_, copyErr := io.Copy(part, file)
	if copyErr != nil {
		return errors.WithStack(copyErr)
	}
	return nil
}

//PostWithJSONApplication func
func (hg *HTTPGateway) PostWithJSONApplication(resourceURL string, jsonObj interface{}) (int, error) {
	//prepare parameters
	data, marErr := json.Marshal(jsonObj)
	if marErr != nil {
		return 0, errors.WithStack(marErr)
	}
	body := new(bytes.Buffer)
	_, writeErr := body.Write(data)
	if writeErr != nil {
		return 0, errors.WithStack(writeErr)
	}
	//do request
	return hg.doRequestWithContentType(resourceURL, "application/json", body)
}

func (hg *HTTPGateway) doRequestWithContentType(resourceURL, contentType string, body io.Reader) (int, error) {
	request, err := http.NewRequest("POST", resourceURL, body)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	request.Header.Add("Content-Type", contentType)
	if err := logRequest(request); err != nil {
		return 0, err
	}
	response, resErr := hg.Client.Do(request)
	if resErr != nil {
		return 0, errors.WithStack(resErr)
	}
	defer response.Body.Close()
	if err := logResponse(response); err != nil {
		return 0, err
	}
	return response.StatusCode, nil
}

func logRequest(request *http.Request) error {
	reader, err := request.GetBody()
	if err != nil {
		return errors.WithStack(err)
	}
	logStringBuilder := new(strings.Builder)
	logStringBuilder.WriteString("Http request:\n")
	logStringBuilder.WriteString("Method ")
	logStringBuilder.WriteString(request.Method)
	logStringBuilder.WriteString(" ")
	logStringBuilder.WriteString(request.URL.String())
	logStringBuilder.WriteString("\n")
	data, readErr := ioutil.ReadAll(reader)
	if readErr != nil {
		return errors.WithStack(readErr)
	}
	logStringBuilder.Write(data)
	logStringBuilder.WriteString("\n")
	log.Println(logStringBuilder)
	return nil
}

func logResponse(response *http.Response) error {
	logStringBuilder := new(strings.Builder)
	logStringBuilder.WriteString("Http response:\n")
	logStringBuilder.WriteString("Http status: ")
	logStringBuilder.WriteString(strconv.Itoa(response.StatusCode))
	logStringBuilder.WriteString("\n")
	data, readErr := ioutil.ReadAll(response.Body)
	if readErr != nil {
		return errors.WithStack(readErr)
	}
	logStringBuilder.Write(data)
	logStringBuilder.WriteString("\n")
	log.Print(logStringBuilder)
	return nil
}
