CREATE TABLE IF NOT EXISTS "user" (
  id SERIAL PRIMARY KEY,
  email VARCHAR(255) UNIQUE,
  name VARCHAR(100),
  sso_id VARCHAR(50) UNIQUE
);

CREATE TABLE IF NOT EXISTS "video" (
  id VARCHAR(36) PRIMARY KEY,
  user_sso_id VARCHAR(50) REFERENCES "user"(sso_id),
  title VARCHAR(255),
  source_type VARCHAR(20),
  source_url TEXT,
  original_path VARCHAR(500),
  dubbed_path VARCHAR(500),
  status VARCHAR(50) DEFAULT 'uploaded',
  transcription TEXT,
  translated_transcription TEXT,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_video_user ON "video"(user_sso_id);
CREATE INDEX IF NOT EXISTS idx_video_status ON "video"(status);
