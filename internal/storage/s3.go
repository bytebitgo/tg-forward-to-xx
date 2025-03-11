package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"github.com/user/tg-forward-to-xx/internal/config"
)

// S3Client S3 客户端
type S3Client struct {
	client        *minio.Client
	bucket        string
	publicBaseURL string
}

// NewS3Client 创建新的 S3 客户端
func NewS3Client() (*S3Client, error) {
	// 创建 MinIO 客户端
	minioClient, err := minio.New(config.AppConfig.S3.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AppConfig.S3.AccessKeyID, config.AppConfig.S3.SecretAccessKey, ""),
		Secure: config.AppConfig.S3.UseSSL,
		Region: config.AppConfig.S3.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 MinIO 客户端失败: %w", err)
	}

	return &S3Client{
		client:        minioClient,
		bucket:        config.AppConfig.S3.Bucket,
		publicBaseURL: config.AppConfig.S3.PublicBaseURL,
	}, nil
}

// UploadFile 上传文件到 S3
func (s *S3Client) UploadFile(reader io.Reader, objectName, contentType string) (string, error) {
	logrus.WithFields(logrus.Fields{
		"bucket":       s.bucket,
		"object_name": objectName,
		"content_type": contentType,
	}).Debug("开始上传文件到 S3")

	// 如果没有提供 content-type，设置默认值
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 上传文件
	_, err := s.client.PutObject(
		context.Background(),
		config.AppConfig.S3.Bucket,
		objectName,
		reader,
		-1,
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return "", fmt.Errorf("上传文件到 S3 失败: %w", err)
	}

	// 构建公共访问 URL
	publicURL := fmt.Sprintf("https://%s/%s/%s",
		s.publicBaseURL,
		s.bucket,
		objectName,
	)

	logrus.WithFields(logrus.Fields{
		"bucket":      s.bucket,
		"object_name": objectName,
		"public_url":  publicURL,
	}).Debug("文件上传成功")

	return publicURL, nil
} 