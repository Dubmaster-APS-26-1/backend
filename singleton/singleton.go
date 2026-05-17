package singleton

import (
	"dubmaster/input"
	"dubmaster/media"
	"dubmaster/pipeline"
	"dubmaster/transcript"
	"dubmaster/translator"
	"dubmaster/user"
	"dubmaster/video"
)

var (
	UserService  = &user.UserService{}
	VideoService = &video.VideoService{}
	Downloader   input.Downloader = &input.YouTube{}
	MediaHandler media.MediaHandler = &media.FfmpegWrapper{}
	Transcripter transcript.Transcripter = transcript.NewWhisper()
	Translator   translator.Translator = &translator.MarianMT{}
	Queue        *pipeline.Queue
)
