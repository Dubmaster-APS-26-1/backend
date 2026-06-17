package pipeline

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"dubmaster/caption"
	"dubmaster/media"
	"dubmaster/transcript"
	"dubmaster/translator"
	"dubmaster/tts"
	"dubmaster/video"
	"dubmaster/ws"

	"github.com/google/uuid"
)

type JobType string

const (
	JobTranscribe JobType = "transcribe"
	JobTranslate  JobType = "translate"
	JobDub        JobType = "dub"
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
	speaker      tts.Speecher
	subtitler    caption.Subtitler
	hub          *ws.Hub
}

func NewQueue(
	videoService *video.VideoService,
	mediaHandler media.MediaHandler,
	transcripter transcript.Transcripter,
	transl translator.Translator,
	speaker tts.Speecher,
	subtitler caption.Subtitler,
	hub *ws.Hub,
) *Queue {
	return &Queue{
		jobs:         make(chan Job, 100),
		videoService: videoService,
		mediaHandler: mediaHandler,
		transcripter: transcripter,
		translator:   transl,
		speaker:      speaker,
		subtitler:    subtitler,
		hub:          hub,
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

func (q *Queue) broadcastStatus(videoID, status string) {
	if q.hub != nil {
		q.hub.Broadcast(videoID, status)
	}
}

func (q *Queue) worker(id int) {
	for job := range q.jobs {
		slog.Info("[QUEUE] Worker processing job", "worker", id, "videoId", job.VideoID, "type", job.Type)
		switch job.Type {
		case JobTranscribe:
			q.processTranscription(job.VideoID)
		case JobTranslate:
			q.processTranslation(job.VideoID)
		case JobDub:
			q.processDub(job.VideoID)
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
	q.broadcastStatus(videoID, "processing")

	pipeDir, _ := filepath.Abs("pipe/")
	os.MkdirAll(pipeDir, 0755)
	audioPath := filepath.Join(pipeDir, v.ID+"_audio.wav")

	err = q.mediaHandler.ExtractAudio(v.OriginalPath, audioPath)
	if err != nil {
		slog.Error("[QUEUE] Error extracting audio", "id", videoID, "error", err)
		q.videoService.UpdateStatus(videoID, "error")
		q.broadcastStatus(videoID, "error")
		os.Remove(audioPath)
		return
	}

	q.videoService.UpdateStatus(videoID, "transcribing")
	q.broadcastStatus(videoID, "transcribing")

	srtContent, err := q.transcripter.Transcript(audioPath)
	os.Remove(audioPath)
	if err != nil {
		slog.Error("[QUEUE] Error transcribing", "id", videoID, "error", err)
		q.videoService.UpdateStatus(videoID, "error")
		q.broadcastStatus(videoID, "error")
		return
	}

	err = q.videoService.UpdateTranscription(videoID, srtContent)
	if err != nil {
		slog.Error("[QUEUE] Error saving transcription", "id", videoID, "error", err)
		q.videoService.UpdateStatus(videoID, "error")
		q.broadcastStatus(videoID, "error")
		return
	}

	q.broadcastStatus(videoID, "transcribed")
	slog.Info("[QUEUE] Transcription completed", "id", videoID)
}

func (q *Queue) processTranslation(videoID string) {
	v, err := q.videoService.GetByID(videoID)
	if err != nil {
		slog.Error("[QUEUE] Error fetching video", "id", videoID, "error", err)
		q.videoService.UpdateStatus(videoID, "error")
		q.broadcastStatus(videoID, "error")
		return
	}

	q.videoService.UpdateStatus(videoID, "translating")
	q.broadcastStatus(videoID, "translating")

	entries := ParseSRT(v.Transcription)
	if len(entries) == 0 {
		slog.Error("[QUEUE] No SRT entries to translate", "id", videoID)
		q.videoService.UpdateStatus(videoID, "error")
		q.broadcastStatus(videoID, "error")
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
		q.broadcastStatus(videoID, "error")
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
		q.broadcastStatus(videoID, "error")
		return
	}

	q.broadcastStatus(videoID, "translated")
	slog.Info("[QUEUE] Translation completed", "id", videoID)
}

func (q *Queue) processDub(videoID string) {
	v, err := q.videoService.GetByID(videoID)
	if err != nil {
		slog.Error("[QUEUE] Error fetching video for dub", "id", videoID, "error", err)
		q.videoService.UpdateStatus(videoID, "error")
		q.broadcastStatus(videoID, "error")
		return
	}

	if v.TranslatedTranscription == "" {
		slog.Error("[QUEUE] No translation available for dub", "id", videoID)
		q.videoService.UpdateStatus(videoID, "error")
		q.broadcastStatus(videoID, "error")
		return
	}

	pipeDir, _ := filepath.Abs("pipe/")
	videoDir := filepath.Join(pipeDir, v.ID)
	os.MkdirAll(videoDir, 0755)

	filesToRemove := make([]string, 0)
	defer func() {
		for _, f := range filesToRemove {
			os.Remove(f)
		}
	}()

	unmergedVideoPath := filepath.Join(videoDir, uuid.NewString()+"_video.mp4")
	unmergedAudioPath := filepath.Join(videoDir, uuid.NewString()+"_audio.mp4")
	filesToRemove = append(filesToRemove, unmergedVideoPath, unmergedAudioPath)

	if err := q.mediaHandler.Unmerge(v.OriginalPath, unmergedVideoPath, unmergedAudioPath); err != nil {
		slog.Error("[QUEUE] Error unmerging video", "id", videoID, "error", err)
		q.videoService.UpdateStatus(videoID, "error")
		q.broadcastStatus(videoID, "error")
		return
	}

	wavAudioPath := filepath.Join(videoDir, uuid.NewString()+".wav")
	filesToRemove = append(filesToRemove, wavAudioPath)

	if err := q.mediaHandler.ConvertToWav(unmergedAudioPath, wavAudioPath); err != nil {
		slog.Error("[QUEUE] Error converting to wav", "id", videoID, "error", err)
		q.videoService.UpdateStatus(videoID, "error")
		q.broadcastStatus(videoID, "error")
		return
	}

	q.videoService.UpdateStatus(videoID, "generating_voice")
	q.broadcastStatus(videoID, "generating_voice")

	dubbedAudio, err := q.generateSegmentedSpeech(v.TranslatedTranscription, wavAudioPath, videoDir)
	if err != nil {
		slog.Error("[QUEUE] Error generating voice", "id", videoID, "error", err)
		q.videoService.UpdateStatus(videoID, "error")
		q.broadcastStatus(videoID, "error")
		return
	}
	filesToRemove = append(filesToRemove, dubbedAudio)

	q.videoService.UpdateStatus(videoID, "rendering")
	q.broadcastStatus(videoID, "rendering")

	videosDir, _ := filepath.Abs("videos/")
	os.MkdirAll(videosDir, 0755)
	outputVideoPath := filepath.Join(videosDir, v.ID+"_dubbed.mp4")

	tmpMerged := filepath.Join(videoDir, uuid.NewString()+"_merged.mp4")
	filesToRemove = append(filesToRemove, tmpMerged)

	if err := q.mediaHandler.Merge(unmergedVideoPath, dubbedAudio, tmpMerged); err != nil {
		slog.Error("[QUEUE] Error merging audio with video", "id", videoID, "error", err)
		q.videoService.UpdateStatus(videoID, "error")
		q.broadcastStatus(videoID, "error")
		return
	}

	subtitlesPath, err := q.subtitler.GenerateSubtitle(tmpMerged)
	if err != nil {
		slog.Warn("[QUEUE] Subtitle generation failed, skipping", "error", err)
		if renameErr := os.Rename(tmpMerged, outputVideoPath); renameErr != nil {
			slog.Error("[QUEUE] Error renaming file", "error", renameErr)
			q.videoService.UpdateStatus(videoID, "error")
			q.broadcastStatus(videoID, "error")
			return
		}
	} else {
		filesToRemove = append(filesToRemove, subtitlesPath)
		if _, err := q.mediaHandler.MergeSubtitle(tmpMerged, subtitlesPath, outputVideoPath); err != nil {
			slog.Warn("[QUEUE] Subtitle merge failed, using without subtitles", "error", err)
			if renameErr := os.Rename(tmpMerged, outputVideoPath); renameErr != nil {
				slog.Error("[QUEUE] Error renaming file", "error", renameErr)
				q.videoService.UpdateStatus(videoID, "error")
				q.broadcastStatus(videoID, "error")
				return
			}
		}
	}

	if err := q.videoService.UpdateDubbedPath(videoID, outputVideoPath); err != nil {
		slog.Error("[QUEUE] Error saving dubbed path", "id", videoID, "error", err)
		q.videoService.UpdateStatus(videoID, "error")
		q.broadcastStatus(videoID, "error")
		return
	}

	q.broadcastStatus(videoID, "finished")
	slog.Info("[QUEUE] Dubbing completed", "id", videoID)
}

func (q *Queue) generateSegmentedSpeech(text, speakerWavPath, outputDir string) (string, error) {
	plainText := q.extractPlainTextFromSRT(text)

	segments := splitTextIntoSegments(plainText)
	slog.Info("[QUEUE] TTS segments prepared", "segment_count", len(segments), "total_chars", len(plainText))
	for i, seg := range segments {
		slog.Info("[QUEUE] TTS segment", "index", i, "chars", len(seg), "preview", truncateString(seg, 80))
	}

	if len(segments) <= 1 {
		return q.speaker.Speech(plainText, speakerWavPath)
	}

	var audioFiles []string
	for idx, seg := range segments {
		if strings.TrimSpace(seg) == "" {
			continue
		}
		outPath, err := q.speaker.Speech(seg, speakerWavPath)
		if err != nil {
			return "", fmt.Errorf("TTS failed on segment %d: %w", idx, err)
		}
		audioFiles = append(audioFiles, outPath)
	}

	if len(audioFiles) == 0 {
		return "", fmt.Errorf("no audio segments generated")
	}
	if len(audioFiles) == 1 {
		return audioFiles[0], nil
	}

	return q.mediaHandler.ConcatAudio(audioFiles, filepath.Join(outputDir, uuid.NewString()+"-final.wav"))
}

func (q *Queue) extractPlainTextFromSRT(srtContent string) string {
	entries := ParseSRT(srtContent)
	if len(entries) == 0 {
		return srtContent
	}
	var texts []string
	for _, e := range entries {
		texts = append(texts, e.Text)
	}
	return strings.Join(texts, " ")
}

func splitTextIntoSegments(text string) []string {
	sentences := strings.Split(text, ".")
	var result []string
	var current string
	const maxSegmentLen = 500

	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if current != "" {
			current += ". "
		}
		current += s
		if len(current) > maxSegmentLen {
			result = append(result, current+".")
			current = ""
		}
	}
	if current != "" {
		result = append(result, current+".")
	}
	return result
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
