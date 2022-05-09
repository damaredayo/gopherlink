package gopherlink

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/damaredayo/goutubedl"
)

func YoutubeToAAC(url string) (aac []byte, info *goutubedl.Info, err error) {

	goutubedl.Path = "yt-dlp"
	res, err := goutubedl.New(context.Background(), url, goutubedl.Options{Type: goutubedl.TypeSingle})
	if err != nil {
		return nil, nil, err
	}
	info = &goutubedl.Info{}
	err = json.Unmarshal(res.RawJSON, &info)
	if err != nil {
		return nil, nil, err
	}

	r, err := res.DownloadAAC(context.Background(), "140", url)
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()

	b := new(bytes.Buffer)
	_, err = io.Copy(b, r)
	aac = b.Bytes()

	return
}

func YoutubeToInfo(url string) (info *goutubedl.Info, err error) {
	goutubedl.Path = "yt-dlp"
	res, err := goutubedl.New(context.Background(), url, goutubedl.Options{Type: goutubedl.TypeSingle})
	if err != nil {
		return nil, err
	}
	info = &goutubedl.Info{}
	err = json.Unmarshal(res.RawJSON, &info)
	if err != nil {
		return nil, err
	}
	return info, err
}
