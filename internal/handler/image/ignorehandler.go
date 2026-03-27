package image

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/image"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetIgnoredImagesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := image.NewIgnoreLogic(r.Context(), svcCtx)
		resp, err := l.GetIgnoredImages()
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, resp)
		} else {
			httpx.WriteJson(w, http.StatusOK, resp)
		}
	}
}

func SetIgnoreImageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := image.NewIgnoreLogic(r.Context(), svcCtx)
		var req types.IgnoreImageReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, &types.SetIgnoreImageResp{
				Code: 400,
				Msg:  "参数错误: " + err.Error(),
			})
			return
		}
		resp, err := l.SetIgnoreImage(&req)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, resp)
		} else {
			httpx.WriteJson(w, http.StatusOK, resp)
		}
	}
}

func UnignoreImageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := image.NewIgnoreLogic(r.Context(), svcCtx)
		var req types.IgnoreImageReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, &types.SetIgnoreImageResp{
				Code: 400,
				Msg:  "参数错误: " + err.Error(),
			})
			return
		}
		resp, err := l.UnignoreImage(&req)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, resp)
		} else {
			httpx.WriteJson(w, http.StatusOK, resp)
		}
	}
}
