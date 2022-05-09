package gopherlink

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"log"

	fluentffmpeg "github.com/damaredayo/go-fluent-ffmpeg"
)

func AacToPCM(in interface{}) (pcm []int16, sampleRate int) {
	sampleRate = 48000

	var bytesReader *bytes.Reader

	switch i := in.(type) {
	case []byte:
		bytesReader = bytes.NewReader(i)
	case *bytes.Reader:
		bytesReader = i
	default:
		return
	}

	pr, pw := io.Pipe()

	cmd := fluentffmpeg.NewCommand("ffmpeg").
		PipeInput(bytesReader).
		OutputFormat("s16le").
		PipeOutput(pw).
		AudioCodec("pcm_s16le").
		AudioRate(48000)

	go func() {
		defer pw.Close()
		err := cmd.Run()
		if err != nil {
			log.Printf("aacToPCM failed: %v", err)
			return
		}
	}()

	b, err := ioutil.ReadAll(pr)
	if err != nil {
		log.Printf("aacToPCM failed: %v", err)
		return
	}
	pcm = make([]int16, len(b)/2)
	buf := bytes.NewReader(b)
	binary.Read(buf, binary.LittleEndian, pcm)
	pr.Close()

	return
}

func reverse(s []int16) []int16 {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func (v *VoiceConnection) SetPCM(pcm []int16) {
	v.pcm = pcm
}
