package singleton

import (
	"dubmaster/caption"
	"dubmaster/config"
	"dubmaster/input"
	"dubmaster/media"
	"dubmaster/pipeline"
	"dubmaster/transcript"
	"dubmaster/translator"
	"dubmaster/tts"
	"dubmaster/user"
	"dubmaster/video"
	"dubmaster/ws"
	"log"
)

var (
	UserService  = &user.UserService{}
	VideoService = &video.VideoService{}
	Downloader   input.Downloader = &input.YouTube{}
	MediaHandler media.MediaHandler = &media.FfmpegWrapper{}
	Transcripter transcript.Transcripter = initTranscript()
	Translator   translator.Translator = &translator.MarianMT{}
	TTS          tts.Speecher = tts.NewCoquiTTS()
	Subtitler    caption.Subtitler = caption.NewStablets()
	Hub          = ws.NewHub()
	Queue        *pipeline.Queue
)

func initTranscript() transcript.Transcripter {
	if config.GroqApiKey != "" {
		log.Println("[SINGLETON] Using Groq Whisper for transcription")
		return transcript.NewGroqWhisper()
	}
	log.Println("[SINGLETON] Using local Whisper for transcription")
	return transcript.NewWhisper()
}
