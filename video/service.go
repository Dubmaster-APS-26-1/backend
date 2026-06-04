package video

import (
	"context"
	"time"

	"dubmaster/database"

	"github.com/google/uuid"
)

type VideoService struct{}

var (
	createVideoSql                   = `INSERT INTO "video" (id, user_sso_id, title, source_type, source_url, original_path, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	getVideoByIdSql                  = `SELECT id, user_sso_id, title, COALESCE(source_type, ''), COALESCE(source_url, ''), original_path, COALESCE(dubbed_path, ''), status, COALESCE(transcription, ''), COALESCE(translated_transcription, ''), created_at, updated_at FROM "video" WHERE id = $1`
	getVideosByUserSql               = `SELECT id, user_sso_id, title, COALESCE(source_type, ''), COALESCE(source_url, ''), original_path, COALESCE(dubbed_path, ''), status, COALESCE(transcription, ''), COALESCE(translated_transcription, ''), created_at, updated_at FROM "video" WHERE user_sso_id = $1 ORDER BY created_at DESC`
	updateStatusSql                  = `UPDATE "video" SET status = $1, updated_at = $2 WHERE id = $3`
	updateTranscriptionSql           = `UPDATE "video" SET transcription = $1, status = $2, updated_at = $3 WHERE id = $4`
	updateTranslatedTranscriptionSql = `UPDATE "video" SET translated_transcription = $1, status = $2, updated_at = $3 WHERE id = $4`
	updateDubbedPathSql              = `UPDATE "video" SET dubbed_path = $1, status = $2, updated_at = $3 WHERE id = $4`
)

func (vs *VideoService) Create(title, sourceType, sourceUrl, originalPath, userSsoId string) (*Video, error) {
	id := uuid.NewString()
	now := time.Now()
	v := &Video{
		ID:           id,
		UserSsoId:    userSsoId,
		Title:        title,
		SourceType:   sourceType,
		SourceUrl:    sourceUrl,
		OriginalPath: originalPath,
		Status:       "uploaded",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err := database.Pool.Exec(context.Background(), createVideoSql,
		v.ID, v.UserSsoId, v.Title, v.SourceType, v.SourceUrl, v.OriginalPath, v.Status, v.CreatedAt, v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (vs *VideoService) GetByID(id string) (*Video, error) {
	var v Video
	var sourceType, sourceUrl string
	err := database.Pool.QueryRow(context.Background(), getVideoByIdSql, id).Scan(
		&v.ID, &v.UserSsoId, &v.Title, &sourceType, &sourceUrl,
		&v.OriginalPath, &v.DubbedPath, &v.Status,
		&v.Transcription, &v.TranslatedTranscription,
		&v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	v.SourceType = sourceType
	v.SourceUrl = sourceUrl
	return &v, nil
}

func (vs *VideoService) GetByUser(userSsoId string) ([]*Video, error) {
	rows, err := database.Pool.Query(context.Background(), getVideosByUserSql, userSsoId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []*Video
	for rows.Next() {
		var v Video
		var sourceType, sourceUrl string
		err := rows.Scan(
			&v.ID, &v.UserSsoId, &v.Title, &sourceType, &sourceUrl,
			&v.OriginalPath, &v.DubbedPath, &v.Status,
			&v.Transcription, &v.TranslatedTranscription,
			&v.CreatedAt, &v.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		v.SourceType = sourceType
		v.SourceUrl = sourceUrl
		videos = append(videos, &v)
	}
	return videos, nil
}

func (vs *VideoService) UpdateStatus(id, status string) error {
	_, err := database.Pool.Exec(context.Background(), updateStatusSql, status, time.Now(), id)
	return err
}

func (vs *VideoService) UpdateTranscription(id, transcription string) error {
	_, err := database.Pool.Exec(context.Background(), updateTranscriptionSql, transcription, "transcribed", time.Now(), id)
	return err
}

func (vs *VideoService) UpdateTranslatedTranscription(id, translated string) error {
	_, err := database.Pool.Exec(context.Background(), updateTranslatedTranscriptionSql, translated, "translated", time.Now(), id)
	return err
}

func (vs *VideoService) UpdateDubbedPath(id, dubbedPath string) error {
	_, err := database.Pool.Exec(context.Background(), updateDubbedPathSql, dubbedPath, "finished", time.Now(), id)
	return err
}
