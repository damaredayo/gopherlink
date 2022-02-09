package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"log"
	"time"

	fluentffmpeg "github.com/damaredayo/go-fluent-ffmpeg"
	"golang.org/x/crypto/nacl/secretbox"
	"gopkg.in/hraban/opus.v2"
)

func (v *VoiceConnection) createOpus(udpclose <-chan struct{}, rate int, size int) {

	if v.udp == nil || udpclose == nil {
		return
	}

	v.wsMutex.Lock()
	v.Ready = true
	v.wsMutex.Unlock()
	defer func() {
		v.wsMutex.Lock()
		v.Ready = false
		v.wsMutex.Unlock()
	}()

	var sequence uint16
	var timestamp uint32
	var recvbuf []byte
	var ok bool
	udpHeader := make([]byte, 12)
	var nonce [24]byte

	udpHeader[0] = 0x80
	udpHeader[1] = 0x78
	binary.BigEndian.PutUint32(udpHeader[8:], v.op2.SSRC)

	ticker := time.NewTicker(time.Millisecond * time.Duration(size/(rate/1000)))
	defer ticker.Stop()

	const sampleRate = 48000
	const channels = 2
	a := 0
	log.Printf("Opening opus sender")
	for {
		select {
		case <-udpclose:
			return
		case <-ticker.C:
			select {
			case recvbuf, ok = <-v.OpusSend:
				a++
				if !ok {
					return
				}

			default:
				continue
			}
		}

		v.Mutex.RLock()
		speaking := v.Speaking
		v.Mutex.RUnlock()
		if !speaking {
			err := v.SetSpeaking(true)
			if err != nil {
				log.Panicln(err)
			}
		}

		binary.BigEndian.PutUint16(udpHeader[2:], sequence)
		binary.BigEndian.PutUint32(udpHeader[4:], timestamp)

		copy(nonce[:], udpHeader)
		v.Mutex.RLock()
		sendbuf := secretbox.Seal(udpHeader, recvbuf, &nonce, &v.op4.SecretKey)
		_, err := v.udp.Write(sendbuf)
		v.Mutex.RUnlock()
		if err != nil {
			log.Println("A", err)
		}

		if (sequence) == 0xFFFF {
			sequence = 0
		} else {
			sequence++
		}

		if (timestamp + uint32(size)) >= 0xFFFFFFFF {
			timestamp = 0
		} else {
			timestamp += uint32(size)
		}
	}
}

func (v *VoiceConnection) musicPlayer(udpclose <-chan struct{}, rate int, size int, pcm []int16) {
	v.Playing = true
	v.paused = false
	const channels = 2

	//pcmbuf := make([]int16, size*2)

	packet_size := size * channels
	bufsize := packet_size * 2 // *2 because []int16 to byte costs 2 bytes per entry

	ticker := time.NewTicker(time.Millisecond * time.Duration(size/(rate/1000)))
	log.Println("ticker", (time.Millisecond * time.Duration(size/(rate/1000))).String())

	enc, err := opus.NewEncoder(48000, 2, opus.AppAudio)
	if err != nil {
		log.Panicf("failed to make opus encoder %v\n", err)
	}
	a := 0
	defer ticker.Stop()

	pcmlen := len(pcm)

	for {
		select {
		case <-udpclose:
			log.Println("music player udpclose called")
			return
		default:
			var perc float64 = float64(a) / float64(pcmlen)
			log.Printf("\rmusic sending %v, %v left to go (%.2f%%)", a, len(pcm), perc*100)
			if v.Reconnecting || v.OpusSend == nil {
				log.Printf("music player: reconnecting... (v.Reconnecting=%v, v.OpusSend=%v", v.Reconnecting, v.OpusSend)
			}
			for {
				if len(pcm) >= packet_size {
					a += packet_size
					pcmbuf := pcm[:packet_size]
					pcm = pcm[packet_size:]

					frameSizeMs := float32(packet_size) / channels * 1000 / 48000
					switch frameSizeMs {
					case 2.5, 5, 10, 20, 40, 60:

					default:
						log.Printf("Illegal frame size: %d bytes (%f ms)", packet_size, frameSizeMs)
					}

					data := make([]byte, bufsize)
					n, err := enc.Encode(pcmbuf, data)
					if err != nil {
						log.Panicf("failed to encode pcm data into opus: %v\n", err)
					}
					for v.Reconnecting || v.OpusSend == nil {
						time.Sleep(100 * time.Millisecond)
						log.Println("reconnecting", v.Reconnecting)
					}

					go func() {
						v.OpusSend <- data[:n]
					}()

					break
				}
				if len(pcm) < packet_size {
					v.Playing = false
					return
				}

			}
		}

		select {
		case <-udpclose:
			v.Playing = false
			log.Println("music player udpclose called")
			return
		case <-ticker.C:
		}

	}
}

func aacToPCM(in interface{}) (pcm []int16, sampleRate int) {
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
