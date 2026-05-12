CREATE TABLE IF NOT EXISTS "user" (
  id SERIAL PRIMARY KEY,
  email VARCHAR(255) UNIQUE,
  name VARCHAR(100),
  sso_id VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS "video" (
  id VARCHAR(36) PRIMARY KEY,
  user_sso_id VARCHAR(50) REFERENCES "user"(sso_id),
  title VARCHAR(255),
  original_path VARCHAR(500),
  status VARCHAR(50) DEFAULT 'uploaded',
  transcription TEXT,
  translated_transcription TEXT,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);
