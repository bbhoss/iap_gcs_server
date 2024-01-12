package main

import (
	"context"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/idtoken"
)

var (
	storageClient *storage.Client
	bucketName    string
)

type loggableRequest struct {
	*http.Request
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	bucketName = os.Getenv("GCS_BUCKET")
	if bucketName == "" {
		log.Fatal("GCS_BUCKET environment variable not set")
	}

	if os.Getenv("IAP_AUDIENCE") == "" {
		log.Fatal("IAP_AUDIENCE environment variable not set")
	}

	var err error
	storageClient, err = storage.NewClient(context.Background())
	if err != nil {
		log.Fatalf("storage.NewClient: %v", err)
	}
	defer storageClient.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", handle)
	logger.Info("listening", slog.String("port", port))

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handle(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	if !validateIAP(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	objectName := strings.TrimPrefix(r.URL.Path, "/")
	if objectName == "" {
		objectName = "index.html"
	}

	if strings.HasSuffix(r.URL.Path, "/") {
		objectName += "index.html"
	}

	reader, err := storageClient.Bucket(bucketName).Object(objectName).NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			if objectName == "index.html" {
				logRequest(r, http.StatusNotFound, err)
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			} else {
				objectName = objectName + "/index.html"
			}

			reader, err = storageClient.Bucket(bucketName).Object(objectName).NewReader(ctx)
			if err != nil {
				logRequest(r, http.StatusNotFound, err)
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
		} else {
			logRequest(r, http.StatusInternalServerError, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
	defer reader.Close()

	if _, err := io.Copy(w, reader); err != nil {
		logRequest(r, http.StatusInternalServerError, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	} else {
		logRequest(r, http.StatusOK, nil)
	}

}

func (r loggableRequest) LogValue() slog.Value {
	// Create a group to hold the loggable fields of the request.
	return slog.GroupValue(
		slog.String("method", r.Method),
		slog.String("url", r.URL.String()),
		slog.String("proto", r.Proto),
		slog.Int("proto_major", r.ProtoMajor),
		slog.Int("proto_minor", r.ProtoMinor),
		slog.Int64("content_length", r.ContentLength),
		slog.Any("transfer_encoding", r.TransferEncoding),
		slog.Bool("close", r.Close),
		slog.String("host", r.Host),
		slog.String("remote_addr", r.RemoteAddr),
		slog.String("request_uri", r.RequestURI),
		// Omitted fields: Body, GetBody, TLS, Cancel, Response, etc., as they are not safe for logging or not applicable.
	)
}

func logRequest(r *http.Request, status int, err error) {
	logger := slog.Default()
	loggableRequest := loggableRequest{r}

	if err != nil {
		logger.Error("failed_request",
			slog.Any("error", err),
			slog.Any("http_request", loggableRequest),
			slog.Int("status", status),
		)
	} else {
		logger.Info("successful_request",
			slog.Any("http_request", loggableRequest),
			slog.Int("status", status),
		)
	}
}

func validateIAP(r *http.Request) bool {

	iapHeader := r.Header.Get("X-Goog-IAP-JWT-Assertion")
	if iapHeader == "" {
		log.Printf("No IAP header found")
		return false
	}

	expectedAudience := os.Getenv("IAP_AUDIENCE")
	ctx := context.Background()
	_, err := idtoken.Validate(ctx, iapHeader, expectedAudience)
	if err != nil {
		log.Printf("Invalid IAP header: %v", err)
		return false
	}
	return true
}
