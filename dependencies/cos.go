package dependencies

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/Xushengqwer/go-common/core"
	"github.com/Xushengqwer/user_hub/config"
	"github.com/google/uuid"
	"github.com/tencentyun/cos-go-sdk-v5"
	"go.uber.org/zap"
)

// COSClientInterface 定义了COS客户端需要实现的方法
type COSClientInterface interface {
	GetClient() *cos.Client // 获取原始的 COS 客户端
	// UploadFile 从 io.Reader 上传文件，并返回其公开可访问的 URL
	UploadFile(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) (string, error)
	// UploadUserAvatar 专门用于上传用户头像，返回头像的公开可访问 URL
	UploadUserAvatar(ctx context.Context, userID string, fileName string, reader io.Reader, size int64) (string, error)
	// DeleteObject 从COS删除一个对象
	DeleteObject(ctx context.Context, objectKey string) error
}

type cosClient struct {
	client              *cos.Client
	sdkBucketURL        *url.URL // SDK 操作时使用的存储桶URL
	publicAccessURLBase *url.URL // 用于拼接最终对象公开访问URL的基础部分
	logger              *core.ZapLogger
	cfg                 *config.COSConfig
}

// InitCOS 初始化腾讯云 COS 客户端
func InitCOS(cfg *config.COSConfig, logger *core.ZapLogger) (COSClientInterface, error) {
	if cfg == nil {
		logger.Error("COS 配置为空")
		return nil, fmt.Errorf("COS 配置不能为nil")
	}
	if cfg.SecretID == "" || cfg.SecretKey == "" || cfg.BucketName == "" || cfg.AppID == "" || cfg.Region == "" {
		logger.Error("COS 配置不完整", zap.Any("配置详情", cfg))
		return nil, fmt.Errorf("COS 配置不完整，缺少关键字段 (SecretID, SecretKey, BucketName, AppID, Region)")
	}

	sdkBucketURLStr := fmt.Sprintf("https://%s-%s.cos.%s.myqcloud.com", cfg.BucketName, cfg.AppID, cfg.Region)
	sdkURL, err := url.Parse(sdkBucketURLStr)
	if err != nil {
		logger.Error("解析 COS 存储桶 SDK 操作 URL 失败", zap.String("url", sdkBucketURLStr), zap.Error(err))
		return nil, fmt.Errorf("解析 COS 存储桶 SDK 操作 URL '%s' 失败: %w", sdkBucketURLStr, err)
	}

	var finalPublicURLBase *url.URL
	if cfg.BaseURL != "" { // 如果配置了 BaseURL (例如CDN或自定义域名或桶的默认公共域名)
		pu, err := url.Parse(cfg.BaseURL)
		if err != nil {
			logger.Error("解析配置的 COS 公共访问 BaseURL 失败", zap.String("提供的BaseURL", cfg.BaseURL), zap.Error(err))
			return nil, fmt.Errorf("解析提供的 COS 公共访问 BaseURL '%s' 失败: %w", cfg.BaseURL, err)
		}
		finalPublicURLBase = pu
		logger.Info("COS 将使用配置的 BaseURL 作为公共访问基础", zap.String("baseURL", cfg.BaseURL))
	} else {
		// 如果没有配置 BaseURL，对于公有读的桶，其标准访问URL结构与SDK操作URL结构一致
		finalPublicURLBase = sdkURL
		logger.Info("COS 未配置 BaseURL，将使用标准存储桶 URL 作为公共访问基础", zap.String("默认公共访问基础URL", finalPublicURLBase.String()))
	}

	sdkClientBaseURL := &cos.BaseURL{BucketURL: sdkURL} // SDK操作用这个
	client := cos.NewClient(sdkClientBaseURL, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  cfg.SecretID,
			SecretKey: cfg.SecretKey,
		},
	})

	logger.Info("COS 客户端初始化成功",
		zap.String("存储桶名称", cfg.BucketName),
		zap.String("AppID", cfg.AppID),
		zap.String("地域", cfg.Region),
		zap.String("SDK操作基础URL", sdkURL.String()),
		zap.String("公共访问基础URL", finalPublicURLBase.String()),
	)

	return &cosClient{
		client:              client,
		sdkBucketURL:        sdkURL,
		publicAccessURLBase: finalPublicURLBase,
		logger:              logger,
		cfg:                 cfg,
	}, nil
}

func (c *cosClient) GetClient() *cos.Client {
	return c.client
}

// buildPublicObjectURL 构建对象的完整公共访问URL
func (c *cosClient) buildPublicObjectURL(objectKey string) string {
	// 确保 c.publicAccessURLBase.Path 以 "/" 结尾，除非它就是根路径 "/"
	basePath := c.publicAccessURLBase.Path
	if basePath != "/" && !strings.HasSuffix(basePath, "/") {
		basePath += "/"
	}
	// 确保 objectKey 不以 "/" 开头
	trimmedObjectKey := strings.TrimPrefix(objectKey, "/")

	// 复制基础URL对象，然后修改其Path部分
	finalURL := *c.publicAccessURLBase
	finalURL.Path = basePath + trimmedObjectKey
	return finalURL.String()
}

// UploadFile 从 io.Reader 上传文件，并返回其公开可访问的 URL
func (c *cosClient) UploadFile(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) (string, error) {
	c.logger.Info("开始上传文件到 COS", zap.String("对象键", objectKey), zap.Int64("文件大小", size), zap.String("内容类型", contentType))
	opts := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType:   contentType,
			ContentLength: size,
		},
		// 对于公有读桶，通常不需要在这里特别设置ACL为public-read，对象会继承桶的权限。
		// 如果桶是私有的，而你又想让这个特定对象公有读，可以设置 ACL: "public-read"
	}

	resp, err := c.client.Object.Put(ctx, objectKey, reader, opts)
	if err != nil {
		c.logger.Error("COS 文件上传 API 调用失败", zap.String("对象键", objectKey), zap.Error(err))
		return "", fmt.Errorf("上传文件 '%s' 到 COS 失败: %w", objectKey, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsgBytes, _ := io.ReadAll(resp.Body)
		errMsg := string(errMsgBytes)
		c.logger.Error("COS 文件上传返回非200状态码",
			zap.String("对象键", objectKey),
			zap.Int("状态码", resp.StatusCode),
			zap.String("响应信息", errMsg),
		)
		return "", fmt.Errorf("COS 文件上传失败，状态码: %d, 响应: %s", resp.StatusCode, errMsg)
	}

	// 构建并返回公开访问URL
	publicURL := c.buildPublicObjectURL(objectKey)
	c.logger.Info("COS 文件上传成功", zap.String("对象键", objectKey), zap.String("公开访问URL", publicURL))
	return publicURL, nil
}

// UploadUserAvatar 专门用于上传用户头像, 返回头像的公开可访问URL
func (c *cosClient) UploadUserAvatar(ctx context.Context, userID string, fileName string, reader io.Reader, size int64) (string, error) {
	ext := filepath.Ext(fileName)
	if ext == "" {
		c.logger.Warn("无法从文件名推断头像扩展名", zap.String("原始文件名", fileName), zap.String("用户ID", userID))
	}
	uniqueFileName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), uuid.New().String(), ext)
	objectKey := fmt.Sprintf("avatars/%s/%s", userID, uniqueFileName)

	var contentType string
	lowerExt := strings.ToLower(ext)
	switch lowerExt {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	default:
		contentType = "application/octet-stream"
		c.logger.Warn("未知的头像文件扩展名，将使用 application/octet-stream 作为 Content-Type", zap.String("扩展名", ext), zap.String("对象键", objectKey))
	}

	c.logger.Info("准备上传用户头像到 COS",
		zap.String("用户ID", userID),
		zap.String("原始文件名", fileName),
		zap.String("COS对象键", objectKey),
		zap.String("内容类型", contentType),
	)
	return c.UploadFile(ctx, objectKey, reader, size, contentType)
}

// DeleteObject 从COS删除一个对象
func (c *cosClient) DeleteObject(ctx context.Context, objectKey string) error {
	c.logger.Info("准备从 COS 删除对象", zap.String("对象键", objectKey))
	resp, err := c.client.Object.Delete(ctx, objectKey)
	if err != nil {
		c.logger.Error("COS 对象删除 API 调用失败", zap.String("对象键", objectKey), zap.Error(err))
		return fmt.Errorf("从 COS 删除对象 '%s' 失败: %w", objectKey, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		errMsgBytes, _ := io.ReadAll(resp.Body)
		errMsg := string(errMsgBytes)
		c.logger.Error("COS 对象删除返回非成功状态码", zap.String("对象键", objectKey), zap.Int("状态码", resp.StatusCode), zap.String("响应信息", errMsg))
		return fmt.Errorf("COS 对象删除失败，状态码: %d, 响应: %s", resp.StatusCode, errMsg)
	}
	c.logger.Info("COS 对象删除成功", zap.String("对象键", objectKey))
	return nil
}
