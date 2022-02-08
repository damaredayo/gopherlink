package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"os"
	"time"

	"github.com/damaredayo/goydl"
	"github.com/xfrr/goffmpeg/transcoder"
	"golang.org/x/crypto/nacl/secretbox"
	"gopkg.in/hraban/opus.v2"
)

func (v *VoiceConnection) createOpus(udp *net.UDPConn, close <-chan int, rate int, size int) {

	if udp == nil || close == nil {
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

	//c, err := oto.NewContext(48000, 2, 2, 16384)
	//if err != nil {
	//	log.Panicln(err)
	//}
	//p := c.NewPlayer()
	a := 0
	log.Println("starting opus...")
	for {
		select {
		case <-close:
			return
		case recvbuf, ok = <-v.OpusSend:
			a++
			if !ok {
				continue
			}
		}

		//p.Write(recvbuf)

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
		v.Mutex.RUnlock()

		select {
		case <-close:
			return
		case <-ticker.C:
			// continue
		}
		_, err := udp.Write(sendbuf)
		log.Printf("sent %v, ok:%v", a, ok)
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

	ticker := time.NewTicker(time.Millisecond * time.Duration(size/(rate/1000)))
	log.Println("ticker", (time.Millisecond * time.Duration(size/(rate/1000))).String())

	//pcmbuf := make([]int16, size*2)

	pcmsize := size * channels
	bufsize := pcmsize * 2 // *2 because []int16 to byte costs 2 bytes per entry

	enc, err := opus.NewEncoder(48000, 2, opus.AppAudio)
	if err != nil {
		log.Panicln("q", err)
	}
	a := 0
	defer ticker.Stop()
	for {
		a++
		log.Println("music sending ", a)
		select {
		case <-udpclose:
			log.Println("music player udpclose called")
			return
		default:
			if v.paused {
				break
			}
			if v.Reconnecting || v.OpusSend == nil {
				log.Printf("music player: reconnecting... (v.Reconnecting=%v, v.OpusSend=%v", v.Reconnecting, v.OpusSend)
			}
			for {
				if len(pcm) >= pcmsize {
					pcmbuf := pcm[:pcmsize]
					pcm = pcm[pcmsize:]

					frameSizeMs := float32(pcmsize) / channels * 1000 / 48000
					switch frameSizeMs {
					case 2.5, 5, 10, 20, 40, 60:

					default:
						log.Printf("Illegal frame size: %d bytes (%f ms)", pcmsize, frameSizeMs)
					}

					data := make([]byte, bufsize)
					n, err := enc.Encode(pcmbuf, data)
					if err != nil {
						log.Panicln("q", err)
					}
					log.Println("sending ", a)
					for v.Reconnecting || v.OpusSend == nil {
						time.Sleep(100 * time.Millisecond)
						log.Println("reconnecting", v.Reconnecting)
					}

					if !v.Reconnecting {
						go func() {
							v.OpusSend <- data[:n]
						}()
					}
					break

				}

			}
		}

		select {
		case <-udpclose:
			v.Playing = false
			log.Println("music player udpclose called")
			return
		case <-ticker.C:
			//fmt.Println("continue: ", <-ticker.C)
		}

	}
}

func aacToPCM() (pcm []int16, sampleRate int) {
	t := new(transcoder.Transcoder)
	err := t.Initialize("./yes.m4a", "pcm.pcm")
	if err != nil {
		log.Println(err)
		return
	}
	t.MediaFile().SetAudioCodec("pcm_s16le")
	t.MediaFile().SetOutputFormat("s16le")
	t.MediaFile().SetAudioRate(48000)
	sampleRate = 48000

	done := t.Run(true)
	progress := t.Output()

	for msg := range progress {
		log.Printf("%+v\n", msg)
	}

	err = <-done
	if err != nil {
		log.Println(err)
		return
	}

	f, err := os.ReadFile("pcm.pcm")
	if err != nil {
		log.Println(err)
		return
	}

	pcm = make([]int16, len(f)/2)
	buf := bytes.NewReader(f)
	binary.Read(buf, binary.LittleEndian, pcm)
	return
}

func youtubeTo(url string) goydl.Info {
	yt := goydl.NewYoutubeDl()
	yt.YoutubeDlPath = "yt-dlp"
	yt.Options.Output.Value = "./yes.m4a"
	yt.Options.ExtractAudio.Value = true
	yt.Options.AudioFormat.Value = "m4a"
	yt.Options.AudioQuality.Value = "0"
	yt.Options.Format.Value = "bestaudio/best"

	cmd, err := yt.Download(url)
	defer cmd.Wait()
	if err != nil {
		log.Fatalln(err)
	}

	return yt.Info

}
