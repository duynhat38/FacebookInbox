package service

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"facebook-inbox/internal/storage"
	"facebook-inbox/internal/utils"
)

// ErrUnsupportedImage is returned when a selected file is not a supported image.
var ErrUnsupportedImage = errors.New("validation.image")

// ImageService validates image selections and copies them into assets/images.
// It never stores image binary data in JSON.
type ImageService struct {
	imagesDir string
}

func NewImageService(p storage.Paths) *ImageService {
	return &ImageService{imagesDir: p.ImagesDir}
}

// Validate reports whether path points to an existing supported image file.
func (s *ImageService) Validate(path string) error {
	if path == "" {
		return nil
	}
	if !utils.IsImageExt(filepath.Ext(path)) {
		return ErrUnsupportedImage
	}
	if !utils.FileExists(path) {
		return fmt.Errorf("image not found: %s", filepath.Base(path))
	}
	return nil
}

// ImportToAssets copies the selected image into assets/images and returns the
// new path. If the file is already inside assets/images it is returned as-is.
func (s *ImageService) ImportToAssets(src string) (string, error) {
	if err := s.Validate(src); err != nil {
		return "", err
	}
	if err := utils.EnsureDir(s.imagesDir); err != nil {
		return "", err
	}

	absSrc, err := filepath.Abs(src)
	if err != nil {
		return "", fmt.Errorf("resolve image path: %w", err)
	}
	dst := filepath.Join(s.imagesDir, filepath.Base(src))
	absDst, err := filepath.Abs(dst)
	if err != nil {
		return "", fmt.Errorf("resolve image dest: %w", err)
	}
	if absSrc == absDst {
		return absDst, nil
	}

	in, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("open image: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return "", fmt.Errorf("create image copy: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return "", fmt.Errorf("copy image: %w", err)
	}
	if err := out.Close(); err != nil {
		return "", fmt.Errorf("finalize image copy: %w", err)
	}
	return dst, nil
}
