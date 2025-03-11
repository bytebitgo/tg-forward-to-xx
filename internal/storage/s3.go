package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"github.com/user/tg-forward-to-xx/config"
)

// S3Client S3 客户端
type S3Client struct {
	client        *minio.Client
	bucket        string
	publicBaseURL string
}

// NewS3Client 创建新的 S3 客户端
func NewS3Client() (*S3Client, error) {
	cfg := config.AppConfig.S3

	// 创建 MinIO 客户端
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 S3 客户端失败: %w", err)
	}

	// 检查 bucket 是否存在
	exists, err := client.BucketExists(context.Background(), cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("检查 bucket 失败: %w", err)
	}

	// 如果 bucket 不存在，创建它
	if !exists {
		err = client.MakeBucket(context.Background(), cfg.Bucket, minio.MakeBucketOptions{
			Region: cfg.Region,
		})
		if err != nil {
			return nil, fmt.Errorf("创建 bucket 失败: %w", err)
		}
		logrus.Infof("创建 bucket: %s", cfg.Bucket)
	}

	return &S3Client{
		client:        client,
		bucket:        cfg.Bucket,
		publicBaseURL: cfg.PublicBaseURL,
	}, nil
}

// UploadFile 上传文件到 S3
func (c *S3Client) UploadFile(reader io.Reader, objectName string, contentType string) (string, error) {
	ctx := context.Background()

	// 生成唯一的对象名称
	timestamp := time.Now().Format("20060102150405")
	objectName = fmt.Sprintf("%s/%s_%s", filepath.Dir(objectName), timestamp, filepath.Base(objectName))

	// 上传文件
	_, err := c.client.PutObject(ctx, c.bucket, objectName, reader, -1,
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", fmt.Errorf("上传文件失败: %w", err)
	}

	// 返回公共访问 URL
	if c.publicBaseURL != "" {
		return fmt.Sprintf("%s/%s", c.publicBaseURL, objectName), nil
	}

	// 如果没有配置公共访问 URL，生成预签名 URL（24小时有效）
	url, err := c.client.PresignedGetObject(ctx, c.bucket, objectName, time.Hour*24, nil)
	if err != nil {
		return "", fmt.Errorf("生成预签名 URL 失败: %w", err)
	}

	return url.String(), nil
} 