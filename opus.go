package gopherlink

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/nacl/secretbox"
	"gopkg.in/hraban/opus.v2"
)

func (v *VoiceConnection) createOpus(udpclose <-chan struct{}, rate int, size int) {

	if v.udp == nil || udpclose == nil {
		return
	}

	v.Mutex.Lock()
	v.Ready = true
	v.Mutex.Unlock()
	defer func() {
		v.Mutex.Lock()
		v.Ready = false
		v.Mutex.Unlock()
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
	v.Reconnecting = false
	err := v.SetSpeaking(true)
	if err != nil {
		log.Panicln(err)
	}
	for {
		select {
		case <-udpclose:
			return
		case <-ticker.C:
			select {
			case recvbuf, ok = <-v.OpusSend:
				a++
				if !ok {
					continue
				}
			}
		}

		binary.BigEndian.PutUint16(udpHeader[2:], sequence)
		binary.BigEndian.PutUint32(udpHeader[4:], timestamp)

		copy(nonce[:], udpHeader)
		v.udpMutex.Lock()
		sendbuf := secretbox.Seal(udpHeader, recvbuf, &nonce, &v.op4.SecretKey)
		_, err = v.udp.Write(sendbuf)
		v.udpMutex.Unlock()
		if err != nil {
			log.Println("failed to write udp", err)
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

func (v *VoiceConnection) MusicPlayer(rate int, size int) {
	v.playerclose = make(chan struct{})

	v.Playing = true
	v.Paused = false
	const channels = 2

	//pcmbuf := make([]int16, size*2)

	packet_size := size * channels
	bufsize := packet_size * 2 // *2 because []int16 to byte costs 2 bytes per entry

	ticker := time.NewTicker(time.Millisecond * time.Duration(size/(rate/1000)))

	enc, err := opus.NewEncoder(48000, 2, opus.AppAudio)
	if err != nil {
		log.Panicf("failed to make opus encoder %v\n", err)
	}
	enc.SetDTX(true)

	v.ByteTrack = 0
	defer ticker.Stop()

	pcmlen := len(v.pcm)

	for {
		select {
		case <-v.playerclose:
			v.Playing = false
			v.NowPlaying = nil
			return
		default:
			if v.Paused {
				time.Sleep(time.Millisecond * 100)
				continue
			}
			for {
				if len(v.pcm) >= packet_size {
					v.ByteTrack += packet_size
					nextByteTrack := v.ByteTrack + packet_size
					if nextByteTrack >= pcmlen {
						nextByteTrack = pcmlen
					}
					pcmbuf := v.pcm[v.ByteTrack:nextByteTrack]

					frameSizeMs := float32(packet_size) / channels * 1000 / 48000
					switch frameSizeMs {
					case 2.5, 5, 10, 20, 40, 60:

					default:
						log.Printf("Illegal frame size: %d bytes (%f ms)", packet_size, frameSizeMs)
					}

					// end of the pcm
					if len(v.pcm) < packet_size || nextByteTrack == pcmlen {
						if v.Loop {
							go v.MusicPlayer(rate, 960)
							return
						}
						err := v.Skip()
						if err != nil {
							if err != ErrNoNextSong {
								log.Println("failed to skip", err)
							}
							v.Playing = false
							v.NowPlaying = nil
							return
						}
						return
					}

					// adjust the volume
					if v.Volume != 1.0 {
						for i := 0; i < len(pcmbuf); i++ {
							pcmbuf[i] = int16(float32(pcmbuf[i]) * v.Volume)
						}
					}

					data := make([]byte, bufsize)
					n, err := enc.Encode(pcmbuf, data)
					if err != nil {
						log.Panicf("failed to encode pcm data into opus: %v\n", err)
					}

					go func() {
						if v.OpusSend == nil {
							v.OpusSend = make(chan []byte, 2)
						}
						if !v.Reconnecting {
							v.OpusSend <- data[:n]
						}
					}()
					break
				}
			}
		}

		select {
		case <-v.playerclose:
			v.Playing = false
			v.NowPlaying = nil
			return
		case <-ticker.C:
		}
	}
}

func (v *VoiceConnection) GetElapsed() int64 {
	if v.NowPlaying == nil {
		return 0
	}
	return int64(v.ByteTrack / 96000)
}

func (v *VoiceConnection) GetDuration() int64 {
	if v.NowPlaying == nil {
		return 0
	}
	return int64(len(v.pcm) / 96000)
}

func (v *VoiceConnection) GetPCMLength() int {
	v.Mutex.Lock()
	defer v.Mutex.Unlock()
	return len(v.pcm)
}

func (v *VoiceConnection) Skip() error {
	if v.NowPlaying == nil {
		return fmt.Errorf("no song playing")
	}

	go func() {
		v.playerclose <- struct{}{}
	}()

	aac, info, err := v.Cache.GetNextSong()
	if err != nil {
		if err != ErrNoNextSong {
			return err
		}
		nextInfo, err := v.Queue.GetNextSong(context.Background())
		if err != nil {
			return err
		}
		aac, info, err = YoutubeToAAC(nextInfo.GetURL())
		if err != nil {
			return err
		}
	}

	pcm, rate := AacToPCM(aac)
	v.pcm = pcm
	v.NowPlaying = &Np{
		GuildId:  v.GuildID,
		Playing:  true,
		Duration: int64(info.Duration),
		Elapsed:  v.GetElapsed(),
		Author:   info.Uploader,
		Title:    info.Title,
	}
	go v.MusicPlayer(rate, 960)

	nextInfo, err := v.Queue.GetNextSong(context.Background())
	if err != nil {
		if err != ErrNoNextSong {
			return err
		}
	}

	err = v.Cache.PreloadSong(nextInfo.GetURL())

	return err
}

func (v *VoiceConnection) Stop() error {
	v.Mutex.Lock()
	defer v.Mutex.Unlock()
	if v.playerclose == nil {
		return fmt.Errorf("no song playing")
	}
	v.NowPlaying = nil
	v.Playing = false
	v.Paused = false
	v.playerclose <- struct{}{}

	return nil
}
