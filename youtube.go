package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"

	"github.com/damaredayo/goutubedl"
)

func youtubeToAAC(url string) (aac []byte, info *goutubedl.Info, err error) {
	goutubedl.Path = "yt-dlp"
	res, err := goutubedl.New(context.Background(), url, goutubedl.Options{Type: goutubedl.TypeSingle})
	if err != nil {
		return nil, nil, err
	}

	log.Printf("downloading YT... %v\n", res.Info)
	log.Printf("downloading YT... \n%v\n", string(res.RawJSON))
	info = &goutubedl.Info{}
	err = json.Unmarshal(res.RawJSON, &info)
	log.Println(err)

	log.Printf("downloading YT... %v\n", info.URL)

	r, err := res.DownloadAAC(context.Background(), "140", url)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	b := new(bytes.Buffer)
	n, err := io.Copy(b, r)
	log.Printf("copied aac %v", n)
	aac = b.Bytes()
	return
}
