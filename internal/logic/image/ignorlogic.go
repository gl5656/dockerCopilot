package image

import (
	"context"
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/module"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type IgnoreLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewIgnoreLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IgnoreLogic {
	return &IgnoreLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetIgnoredImages 获取所有忽略更新的镜像ID列表
func (l *IgnoreLogic) GetIgnoredImages() (resp *types.IgnoredImagesResp, err error) {
	imageData := module.NewImageCheck()
	ignoredList := imageData.GetIgnoredImages()
	return &types.IgnoredImagesResp{
		Code: 200,
		Msg:  "success",
		Data: ignoredList,
	}, nil
}

// SetIgnoreImage 设置镜像忽略状态
func (l *IgnoreLogic) SetIgnoreImage(req *types.IgnoreImageReq) (resp *types.SetIgnoreImageResp, err error) {
	imageData := module.NewImageCheck()
	imageData.IgnoreImage(req.ImageId)
	imageData.SaveIgnored()
	return &types.SetIgnoreImageResp{
		Code: 200,
		Msg:  "镜像已设为忽略更新",
	}, nil
}

// UnignoreImage 取消镜像忽略状态
func (l *IgnoreLogic) UnignoreImage(req *types.IgnoreImageReq) (resp *types.SetIgnoreImageResp, err error) {
	imageData := module.NewImageCheck()
	imageData.UnignoreImage(req.ImageId)
	imageData.SaveIgnored()
	return &types.SetIgnoreImageResp{
		Code: 200,
		Msg:  "镜像已取消忽略",
	}, nil
}
