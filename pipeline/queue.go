package pipeline

import (
	"log/slog"
	"os"
	"path/filepath"

	"dubmaster/media"
	"dubmaster/transcript"
	"dubmaster/translator"
	"dubmaster/video"
)

type JobType string

const (
	JobTranscribe JobType = "transcribe"
	JobTranslate  JobType = "translate"
)

type Job struct {
	VideoID string
	Type    JobType
}

type Queue struct {
	jobs         chan Job
	videoService *video.VideoService
	mediaHandler media.MediaHandler
	transcripter transcript.Transcripter
	translator   translator.Translator
}

func NewQueue(
	videoService *video.VideoService,
	mediaHandler media.MediaHandler,
	transcripter transcript.Transcripter,
	transl translator.Translator,
) *Queue {
	return &Queue{
		jobs:         make(chan Job, 100),
		videoService: videoService,
		mediaHandler: mediaHandler,
		transcripter: transcripter,
		translator:   transl,
	}
}

func (q *Queue) Start(workers int) {
	for i := 0; i < workers; i++ {
		go q.worker(i)
	}
	slog.Info("[QUEUE] Started processing queue", "workers", workers)
}

func (q *Queue) Enqueue(job Job) {
	q.jobs <- job
	slog.Info("[QUEUE] Job enqueued", "videoId", job.VideoID, "type", job.Type)
}

func (q *Queue) worker(id int) {
	for job := range q.jobs {
		slog.Info("[QUEUE] Worker processing job", "worker", id, "videoId", job.VideoID, "type", job.Type)
		switch job.Type {
		case JobTranscribe:
			q.processTranscription(job.VideoID)
		case JobTranslate:
			q.processTranslation(job.VideoID)
		}
	}
}

func (q *Queue) processTranscription(videoID string) {
	v, err := q.videoService.GetByID(videoID)
	if err != nil {
		slog.Error("[QUEUE] Error fetching video", "id", videoID, "error", err)
		q.videoService.UpdateStatus(videoID, "error")
		return
	}

	q.videoService.UpdateStatus(videoID, "processing")

	pipeDir, _ := filepath.Abs("pipe/")
	os.MkdirAll(pipeDir, 0755)
	audioPath := filepath.Join(pipeDir, v.ID+"_audio.wav")

	err = q.mediaHandler.ExtractAudio(v.OriginalPath, audioPath)
	if err != nil {
		slog.Error("[QUEUE] Error extracting audio", "id", videoID, "error", err)
		q.videoService.UpdateStatus(videoID, "error")
		os.Remove(audioPath)
		return
	}

	q.videoService.UpdateStatus(videoID, "transcribing")

	srtContent, err := q.transcripter.Transcript(audioPath)
	os.Remove(audioPath)
	if err != nil {
		slog.Error("[QUEUE] Error transcribing", "id", videoID, "error", err)
		q.videoService.UpdateStatus(videoID, "error")
		return
	}

	err = q.videoService.UpdateTranscription(videoID, srtContent)
	if err != nil {
		slog.Error("[QUEUE] Error saving transcription", "id", videoID, "error", err)
		q.videoService.UpdateStatus(videoID, "error")
		return
	}

	slog.Info("[QUEUE] Transcription completed", "id", videoID)
}

func (q *Queue) processTranslation(videoID string) {
	v, err := q.videoService.GetByID(videoID)
	if err != nil {
		slog.Error("[QUEUE] Error fetching video", "id", videoID, "error", err)
		q.videoService.UpdateStatus(videoID, "error")
		return
	}

	q.videoService.UpdateStatus(videoID, "translating")

	entries := ParseSRT(v.Transcription)
	if len(entries) == 0 {
		slog.Error("[QUEUE] No SRT entries to translate", "id", videoID)
		q.videoService.UpdateStatus(videoID, "error")
		return
	}

	texts := make([]string, len(entries))
	for i, e := range entries {
		texts[i] = e.Text
	}

	translatedTexts, err := q.translator.TranslateBatch(texts)
	if err != nil {
		slog.Error("[QUEUE] Error translating", "id", videoID, "error", err)
		q.videoService.UpdateStatus(videoID, "error")
		return
	}

	for i := range entries {
		if i < len(translatedTexts) {
			entries[i].Text = translatedTexts[i]
		}
	}

	translatedSRT := BuildSRT(entries)
	err = q.videoService.UpdateTranslatedTranscription(videoID, translatedSRT)
	if err != nil {
		slog.Error("[QUEUE] Error saving translation", "id", videoID, "error", err)
		q.videoService.UpdateStatus(videoID, "error")
		return
	}

	slog.Info("[QUEUE] Translation completed", "id", videoID)
}
