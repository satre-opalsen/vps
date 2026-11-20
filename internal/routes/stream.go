package routes

import (
	"EverythingSuckz/fsb/internal/bot"
	"EverythingSuckz/fsb/internal/stream"
	"EverythingSuckz/fsb/internal/types"
	"EverythingSuckz/fsb/internal/utils"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gotd/td/tg"
	range_parser "github.com/quantumsheep/range-parser"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

var log *zap.Logger

func (e *allRoutes) LoadHome(r *Route) {
	log = e.log.Named("Stream")
	defer log.Info("Loaded stream route")

	paths := []string{
		"/stream/:messageID",
		"/telegram/:messageID",
		"/telegram/:messageID/stream",
		"/telegram/:messageID/dl",
		"/telegram/:messageID/dl=true",
		"/telegram/:messageID/download",
	}

	for _, p := range paths {
		r.Engine.GET(p, getStreamRoute)
		r.Engine.HEAD(p, getStreamRoute)
	}
}

func getStreamRoute(ctx *gin.Context) {
	w := ctx.Writer
	r := ctx.Request

	messageIDParm := ctx.Param("messageID")
	var messageID int
	if len(messageIDParm) == 6 {
		messageID = utils.DecodeMessageID(messageIDParm)
	}
	if messageID == 0 {
		var err error
		messageID, err = strconv.Atoi(messageIDParm)
		if err != nil {
			http.Error(w, "invalid message id", http.StatusBadRequest)
			return
		}
	}

	worker := bot.GetNextWorker()

	file, err := utils.TimeFuncWithResult(log, "FileFromMessage", func() (*types.File, error) {
		return utils.FileFromMessage(ctx, worker.Client, messageID)
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	authHash := ctx.Query("hash")
	if authHash != "" {
		expectedHash := utils.PackFile(
			file.FileName,
			file.FileSize,
			file.MimeType,
			file.ID,
		)
		if !utils.CheckHash(authHash, expectedHash) {
			http.Error(w, "invalid hash", http.StatusBadRequest)
			return
		}
	}

	safeFileName := strings.ReplaceAll(file.FileName, "\"", "_")
	if safeFileName == "" {
		safeFileName = fmt.Sprintf("file_%d", messageID)
	}

	// for photo messages
	if file.FileSize == 0 {
		res, err := worker.Client.API().UploadGetFile(ctx, &tg.UploadGetFileRequest{
			Location: file.Location,
			Offset:   0,
			Limit:    1024 * 1024,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		result, ok := res.(*tg.UploadFile)
		if !ok {
			http.Error(w, "unexpected response", http.StatusInternalServerError)
			return
		}
		fileBytes := result.GetBytes()
		ctx.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", safeFileName))
		if r.Method != "HEAD" {
			ctx.Data(http.StatusOK, file.MimeType, fileBytes)
		}
		return
	}

	ctx.Header("Accept-Ranges", "bytes")
	var start, end int64
	rangeHeader := r.Header.Get("Range")

	if rangeHeader == "" {
		start = 0
		end = file.FileSize - 1
		w.WriteHeader(http.StatusOK)
	} else {
		ranges, err := range_parser.Parse(file.FileSize, r.Header.Get("Range"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		start = ranges[0].Start
		end = ranges[0].End
		ctx.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, file.FileSize))
		log.Info("Content-Range", zap.Int64("start", start), zap.Int64("end", end), zap.Int64("fileSize", file.FileSize))
		w.WriteHeader(http.StatusPartialContent)
	}

	contentLength := end - start + 1
	mimeType := file.MimeType

	if mimeType == "" || mimeType == "application/octet-stream" {
		if ext := filepath.Ext(safeFileName); ext != "" {
			if guessed := mime.TypeByExtension(ext); guessed != "" {
				mimeType = guessed
			}
		}
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	ctx.Header("Content-Type", mimeType)
	ctx.Header("Content-Length", strconv.FormatInt(contentLength, 10))

	disposition := "inline"
	cleanPath := strings.TrimSuffix(r.URL.Path, "/")
	if ctx.Query("d") == "true" || ctx.Query("dl") == "true" ||
		strings.HasSuffix(cleanPath, "/dl") ||
		strings.HasSuffix(cleanPath, "/dl=true") ||
		strings.HasSuffix(cleanPath, "/download") {
		disposition = "attachment"
	}

	ctx.Header("Content-Disposition", fmt.Sprintf("%s; filename=\"%s\"", disposition, safeFileName))

	if r.Method != "HEAD" {
		pipe, err := stream.NewStreamPipe(ctx, worker.Client, file.Location, start, end, log)
		if err != nil {
			log.Error("Failed to create stream pipe", zap.Error(err))
			return
		}
		defer pipe.Close()
		if _, err := io.CopyN(w, pipe, contentLength); err != nil {
			if !utils.IsClientDisconnectError(err) {
				log.Error("Error while copying stream", zap.Error(err))
			}
		}
	}
}
