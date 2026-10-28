package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"

	"business_services/internal/storage"
)

func main() {
	_ = godotenv.Load()

	st, err := storage.NewStorage(storage.Config{
		Endpoint:  env("MINIO_ENDPOINT", "localhost:9000"),
		AccessKey: env("MINIO_ACCESS_KEY", "minioadmin"),
		SecretKey: env("MINIO_SECRET_KEY", "minioadmin"),
		Bucket:    env("MINIO_BUCKET", "metoda-media"),
	})
	if err != nil {
		log.Fatal(err)
	}

	files, _ := filepath.Glob("resources/seed/*")
	for _, p := range files {
		f, err := os.Open(p)
		if err != nil {
			log.Println(err)
			continue
		}

		info, _ := f.Stat()
		ext := strings.ToLower(filepath.Ext(p))
		ct := contentType(ext)
		key := "seed/" + filepath.Base(p)

		if err := st.Upload(context.Background(), key, f, info.Size(), ct); err != nil {
			log.Printf("upload %s: %v", p, err)
		} else {
			log.Printf("загружено: %s", key)
		}
		f.Close()
	}
}

func contentType(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	}
	return "application/octet-stream"
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
