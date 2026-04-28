package singleton

import (
	"dubmaster/input"
	"dubmaster/user"
)

var (
	UserService = &user.UserService{}
	Downloader  input.Downloader = &input.YouTube{}
)