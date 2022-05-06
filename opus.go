package main

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
	log.Printf("Opening opus sender")
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
					return
				}

			default:
				continue
			}
		}

		binary.BigEndian.PutUint16(udpHeader[2:], sequence)
		binary.BigEndian.PutUint32(udpHeader[4:], timestamp)

		copy(nonce[:], udpHeader)
		v.Mutex.Lock()
		sendbuf := secretbox.Seal(udpHeader, recvbuf, &nonce, &v.op4.SecretKey)
		_, err = v.udp.Write(sendbuf)
		v.Mutex.Unlock()
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

func (v *VoiceConnection) musicPlayer(rate int, size int) {
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
	enc.SetDTX(true)

	v.ByteTrack = 0
	defer ticker.Stop()

	pcmlen := len(v.pcm)

	for {
		select {
		case <-v.playerclose:
			log.Println("music player close called")
			return
		default:
			var perc float64 = float64(v.ByteTrack) / float64(pcmlen)
			if v.paused {
				fmt.Printf("\r[PAUSED] (%v): %v | %.2fMB/%.2fMB (%.2f%%)",
					v.GuildID,
					v.NowPlaying.Author+" - "+v.NowPlaying.Title,
					float64(v.ByteTrack)/(1<<20),
					float64(pcmlen)/(1<<20),
					perc*100)
				continue
			}
			fmt.Printf("\r[PLAYING] (%v): %v | %.2fMB/%.2fMB (%.2f%%)",
				v.GuildID,
				v.NowPlaying.Author+" - "+v.NowPlaying.Title,
				float64(v.ByteTrack)/(1<<20),
				float64(pcmlen)/(1<<20),
				perc*100)
			if v.Reconnecting || v.OpusSend == nil {
				log.Printf("music player: reconnecting... (v.Reconnecting=%v, v.OpusSend=%v", v.Reconnecting, v.OpusSend)
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
					if len(v.pcm) < packet_size || nextByteTrack == pcmlen {
						for {
							v.Playing = false
							info, err := v.Queue.GetNextSong(context.Background())
							aac, _, err := youtubeToAAC(info.GetURL())
							if err != nil {
								if err == ErrNoSongFound {
									return
								}
								continue
							}

							if !v.Reconnecting {
								pcm, rate := aacToPCM(aac)
								v.pcm = pcm
								v.playerclose = make(chan struct{})
								go v.musicPlayer(rate, 960)
								return
							}
							log.Println("failed to get next song", err)
						}
					}
					v.Volume = 100
					for i, p := range pcmbuf {
						pcmbuf[i] = p - v.Volume
					}

					data := make([]byte, bufsize)
					n, err := enc.Encode(pcmbuf, data)
					if err != nil {
						log.Panicf("failed to encode pcm data into opus: %v\n", err)
					}
					for v.Reconnecting {
						time.Sleep(100 * time.Millisecond)
						log.Println("reconnecting", v.Reconnecting)
					}

					go func() {
						if v.OpusSend == nil {
							v.OpusSend = make(chan []byte, 2)
						}
						v.OpusSend <- data[:n]
					}()
					break
				}
			}
		}

		select {
		case <-v.playerclose:
			v.Playing = false
			log.Println("music player close called")
			return
		case <-ticker.C:
		}

	}
}
