package module

import (
	"crypto/tls"
	"errors"
	"fmt"
	ref "github.com/distribution/reference"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/sysx"
	"io"
	"net"
	"net/http"
	url2 "net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// ImageCheckList 检查更新处理后的镜像列表
type ImageCheckList struct {
	NeedUpdate bool
}

type ImageUpdateData struct {
	Data map[string]ImageCheckList
	mu   sync.RWMutex
	// 新增：忽略更新的镜像列表，key 为镜像 ID
	IgnoredImages map[string]bool
	ignoreFile    string
}

const ContentDigestHeader = "Docker-Content-Digest"

func NewImageCheck() *ImageUpdateData {
	data := &ImageUpdateData{
		Data:          map[string]ImageCheckList{},
		IgnoredImages: make(map[string]bool),
		ignoreFile:    getIgnoreFilePath(),
	}
	// 启动时加载忽略列表
	data.LoadIgnored()
	return data
}

func getIgnoreFilePath() string {
	return "/app/data/ignore_images.txt"
}

func (i *ImageUpdateData) LoadIgnored() {
	i.mu.Lock()
	defer i.mu.Unlock()

	// 确保目录存在
	dir := "/app/data"
	if err := os.MkdirAll(dir, 0644); err != nil {
		logx.Error("创建忽略文件目录失败: " + err.Error())
		return
	}

	data, err := os.ReadFile(i.ignoreFile)
	if err != nil {
		if !os.IsNotExist(err) {
			logx.Error("读取忽略列表失败: " + err.Error())
		}
		return
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			i.IgnoredImages[line] = true
		}
	}
	logx.Info("已加载忽略镜像列表，共 " + string(rune(len(i.IgnoredImages))) + " 个")
}

func (i *ImageUpdateData) SaveIgnored() {
	i.mu.Lock()
	defer i.mu.Unlock()

	dir := "/app/data"
	if err := os.MkdirAll(dir, 0644); err != nil {
		logx.Error("创建忽略文件目录失败: " + err.Error())
		return
	}

	var lines []string
	for id := range i.IgnoredImages {
		lines = append(lines, id)
	}

	content := strings.Join(lines, "\n")
	if err := os.WriteFile(i.ignoreFile, []byte(content), 0644); err != nil {
		logx.Error("保存忽略列表失败: " + err.Error())
		return
	}
	logx.Info("已保存忽略镜像列表")
}

// IgnoreImage 将镜像ID加入忽略列表
func (i *ImageUpdateData) IgnoreImage(imageId string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.IgnoredImages[imageId] = true
}

// UnignoreImage 将镜像ID从忽略列表移除
func (i *ImageUpdateData) UnignoreImage(imageId string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	delete(i.IgnoredImages, imageId)
}

// IsIgnored 检查镜像是否被忽略
func (i *ImageUpdateData) IsIgnored(imageId string) bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.IgnoredImages[imageId]
}

// GetIgnoredImages 获取所有忽略的镜像ID列表
func (i *ImageUpdateData) GetIgnoredImages() []string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	result := make([]string, 0, len(i.IgnoredImages))
	for id := range i.IgnoredImages {
		result = append(result, id)
	}
	return result
}

func (i *ImageUpdateData) CheckUpdate(imageList []types.Image) {
	for _, image := range imageList {
		// ★ 新增：跳过忽略的镜像
		if i.IsIgnored(image.ID) {
			logx.Info("跳过忽略的镜像: " + image.ImageName + ":" + image.ImageTag)
			continue
		}

		if strings.Contains(image.ImageName, "0nlylty/dockercopilot") {
			continue
		}
		i.checkSingleImage(image)
	}
}

func (i *ImageUpdateData) checkSingleImage(image types.Image) {
	token, err := GetToken(image, "")
	if err != nil {
		logx.Error("获取token失败或者无需获取token，继续尝试检查" + err.Error())
	}
	digestURL, err := BuildManifestURL(image)
	if err != nil {
		logx.Error("获取digestURL失败" + err.Error())
		return
	}
	remoteDigest, err := GetDigest(digestURL, token)
	if err != nil {
		logx.Error("获取digest失败" + err.Error())
		return
	}
	if len(image.RepoDigests) == 0 {
		logx.Error("未在本地获取到repoDigest" + image.ImageName + ":" + image.ImageTag)
		return
	}
	needUpdate := false
	for _, localRepoDigests := range image.RepoDigests {
		localDigest := strings.Split(localRepoDigests, "@")[1]
		if remoteDigest != localDigest {
			if remoteDigest == "" || localDigest == "" {
				logx.Error("Digest为空" + image.ImageName + ":" + image.ImageTag)
				continue
			}
			logx.Info(image.ImageName + ":" + image.ImageTag + " need update")
			logx.Infof("localDigest: %s, remoteDigest: %s", localDigest, remoteDigest)
			needUpdate = true
		} else {
			logx.Info(image.ImageName + ":" + image.ImageTag + " not need update")
			needUpdate = false
		}
	}
	i.Data[image.ID] = ImageCheckList{NeedUpdate: needUpdate}
}

func BuildManifestURL(image types.Image) (string, error) {
	normalizedRef, err := ref.ParseDockerRef(image.ImageName + ":" + image.ImageTag)
	if err != nil {
		return "", err
	}
	normalizedTaggedRef, isTagged := normalizedRef.(ref.NamedTagged)
	if !isTagged {
		return "", errors.New("镜像无tag" + normalizedRef.String())
	}

	host, ErrGetRegistryAddress := GetRegistryAddress(normalizedTaggedRef.Name())
	img, tag := ref.Path(normalizedTaggedRef), normalizedTaggedRef.Tag()

	if ErrGetRegistryAddress != nil {
		return "", ErrGetRegistryAddress
	}

	url := url2.URL{
		Scheme: "https",
		Host:   host,
		Path:   fmt.Sprintf("/v2/%s/manifests/%s", img, tag),
	}
	return url.String(), nil
}

func GetDigest(url string, token string) (string, error) {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, _ := http.NewRequest("HEAD", url, nil)

	if token != "" {
		req.Header.Add("Authorization", token)
	}
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v1+json")
	req.Header.Add("Accept", "application/vnd.oci.image.index.v1+json")

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logx.Error("GetDigest关闭body失败" + err.Error())
		}
	}(res.Body)

	if res.StatusCode != 200 {
		wwwAuthHeader := res.Header.Get("www-authenticate")
		if wwwAuthHeader == "" {
			wwwAuthHeader = "not present"
		}
		return "", fmt.Errorf("registry responded to head request with %q, auth: %q", res.Status, wwwAuthHeader)
	}
	return res.Header.Get(ContentDigestHeader), nil
}

// Hostname 返回主机名
func Hostname() string {
	return sysx.Hostname()
}
