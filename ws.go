package gopherlink

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var ErrOkIsFalse = errors.New("ok is false")

var ErrWSNotNil = errors.New("websocket not nil")
var ErrVoiceWSNil = errors.New("nil voice websocket")
var ErrUDPOpen = errors.New("udp connection already open")
var ErrNilClose = errors.New("nil close channel")
var ErrEmptyEndpoint = errors.New("empty endpoint")
var ErrUDPSmallPacket = errors.New("recv udp packet is small")

type voiceChannelJoinData struct {
	GuildID   *string `json:"guild_id"`
	ChannelID *string `json:"channel_id"`
	SelfMute  bool    `json:"self_mute"`
	SelfDeaf  bool    `json:"self_deaf"`
}

type voiceChannelJoinOp struct {
	Op   int                  `json:"op"`
	Data voiceChannelJoinData `json:"d"`
}

type voiceHandshakeData struct {
	ServerID  string `json:"server_id"`
	UserID    string `json:"user_id,omitempty"`
	SessionID string `json:"session_id"`
	Token     string `json:"token"`
}
type voiceHandshakeOp struct {
	Op   int                `json:"op"`
	Data voiceHandshakeData `json:"d"`
}

type voiceUDPOp struct {
	Op   int       `json:"op"`
	Data voiceUDPD `json:"d"`
}

type voiceUDPD struct {
	Protocol string       `json:"protocol"`
	Data     voiceUDPData `json:"data"`
}

type voiceUDPData struct {
	Address string `json:"address"`
	Port    uint16 `json:"port"`
	Mode    string `json:"mode"`
}

type voiceHeartbeatOp struct {
	Op   int `json:"op"`
	Data int `json:"d"`
}
type Op2 struct {
	SSRC  uint32   `json:"ssrc"`
	Port  int      `json:"port"`
	Modes []string `json:"modes"`
	IP    string   `json:"ip"`
}

type Op4 struct {
	SecretKey [32]byte `json:"secret_key"`
	Mode      string   `json:"mode"`
}

type Op8 struct {
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
}

type Np struct {
	GuildId  string
	Playing  bool
	Duration int64
	Elapsed  int64
	Author   string
	Title    string
}

type VoiceConnection struct {
	Mutex sync.RWMutex

	Ready bool

	UserID    string
	GuildID   string
	ChannelID string

	Deaf     bool
	Mute     bool
	Speaking bool

	Connected    bool
	Reconnecting bool

	SessionID string
	Token     string
	Endpoint  string

	NowPlaying *Np
	Queue      *Queue

	pcm       []int16
	ByteTrack int
	OpusSend  chan []byte

	Playing bool
	Paused  bool
	Volume  float32
	Loop    bool

	op4 Op4
	op2 Op2

	close       chan struct{}
	udpclose    chan struct{}
	playerclose chan struct{}

	ws      *websocket.Conn
	wsMutex sync.Mutex
	udp     *net.UDPConn
}

type Event struct {
	Operation int             `json:"op"`
	Sequence  int64           `json:"s"`
	Type      string          `json:"t"`
	RawData   json.RawMessage `json:"d"`
}

func (v *VoiceConnection) Open() (err error) {
	if v.ws != nil {
		v.wsMutex.Lock()
		v.ws.Close()
		v.ws = nil
		v.wsMutex.Unlock()
	}
	if err := v.makeWS(); err != nil {
		return err
	}

	if v.Reconnecting {
		data := voiceHandshakeOp{7,
			voiceHandshakeData{
				ServerID:  v.GuildID,
				SessionID: v.SessionID,
				UserID:    v.UserID,
				Token:     v.Token},
		}
		v.wsMutex.Lock()
		err = v.ws.WriteJSON(data)
		v.wsMutex.Unlock()
		if err != nil {
			return err
		}

	} else {
		data := voiceHandshakeOp{0, voiceHandshakeData{v.GuildID, v.UserID, v.SessionID, v.Token}}
		v.wsMutex.Lock()
		err = v.ws.WriteJSON(data)
		v.wsMutex.Unlock()
		if err != nil {
			return err
		}
	}

	go v.initListener(v.ws, v.close)

	return
}

func (v *VoiceConnection) makeWS() (err error) {
	v.Mutex.Lock()
	defer v.Mutex.Unlock()

	if v.udp == nil {
		v.udpclose = make(chan struct{})
	}

	wsUrl := "wss://" + strings.TrimSuffix(v.Endpoint, ":80")
	v.ws, _, err = websocket.DefaultDialer.Dial(wsUrl, nil)

	if err != nil {
		return err
	}

	v.close = make(chan struct{})

	return nil
}

func (v *VoiceConnection) initListener(wsConn *websocket.Conn, close <-chan struct{}) {
	for {
		_, msg, err := v.ws.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, 4014) {
				v.Mutex.Lock()
				v.ws = nil
				v.Mutex.Unlock()

				v.Close()

				return
			}

			if websocket.IsCloseError(err, 4006) {
				ce, ok := err.(*websocket.CloseError)
				if ok {
					log.Printf("websocket failed: %v - %v", ce.Code, ce.Text)
					v.Close()
				}
				v.Mutex.Lock()
				v.ws = nil
				v.Mutex.Unlock()

				v.Close()

				return
			}

			if websocket.IsCloseError(err, 1000) {
				v.Reconnecting = true
				v.Close()
				v.ws.Close()
				v.Open()

				return
			} else {
				return
			}
		}
		go v.opcodeHandler(msg)
		select {
		case <-v.close:
			return
		default:
		}
	}
}

func (v *VoiceConnection) opcodeHandler(msg []byte) {
	var e Event
	if err := json.Unmarshal(msg, &e); err != nil {
		return
	}

	switch e.Operation {
	case 1:
		return
	case 2: // voice ready
		v.op2 = Op2{}
		if err := json.Unmarshal(e.RawData, &v.op2); err != nil {
			log.Printf("error: %v\n", err)
			return
		}

		if v.OpusSend == nil {
			v.OpusSend = make(chan []byte, 2)
		}

		if v.Reconnecting {
			v.Mutex.Lock()
			v.udpClose()
			udp, err := v.udpOpen()
			v.Mutex.Unlock()
			if err != nil {
				log.Printf("error: %v\n", err)
				return
			}
			v.udp = udp

		} else {
			if v.udp != nil {
				v.udp.Close()
				v.udp = nil
			}

			udp, err := v.udpOpen()
			v.udp = udp
			if err != nil {
				log.Printf("error: %v\n", err)
				return
			}
		}
		v.Reconnecting = false

		go v.createOpus(v.udpclose, 48000, 960)

	case 3: // heartbeat
		return

	case 4: // udp secret
		v.Mutex.Lock()
		defer v.Mutex.Unlock()

		v.op4 = Op4{}
		if err := json.Unmarshal(e.RawData, &v.op4); err != nil {
			log.Printf("error: %v\n", err)
			return
		}
		return

	case 5: // voice update
		return
	case 6:
		var op6 voiceHeartbeatOp
		if err := json.Unmarshal(e.RawData, &op6); err != nil {
			log.Printf("error: %v\n", err)
			return
		}
		return

	case 8: // hello
		var op8 Op8
		if err := json.Unmarshal(e.RawData, &op8); err != nil {
			log.Printf("error: %v\n", err)
			return
		}

		v.Reconnecting = false
		v.heartbeatInit(op8.HeartbeatInterval)
		return
	case 9: // welcome back :3
		v.Reconnecting = false
	default: // unknown
		log.Printf("unknown op, dumping event data: %v\n", string(e.RawData))
	}

	return
}

func (v *VoiceConnection) Close() {
	v.Mutex.Lock()
	defer v.Mutex.Unlock()

	v.Ready, v.Speaking = false, false

	if v.ws != nil {
		err := v.ws.Close()
		if err != nil {
			log.Fatalln(err)
		}
		close(v.close)
		v.close = nil
	}
}

func (v *VoiceConnection) udpClose() {
	v.Mutex.Lock()
	defer v.Mutex.Unlock()

	v.Ready, v.Speaking = false, false

	if v.udp != nil {
		err := v.udp.Close()
		if err != nil {
			log.Fatalln(err)
		}
		close(v.udpclose)
		v.udpclose = nil
	}
}
func (v *VoiceConnection) heartbeatInit(i time.Duration) {
	if v.close == nil || v.ws == nil {
		return
	}
	ticker := time.NewTicker(i*time.Millisecond - time.Second)
	// first heartbeat manually, then the ticker will take over
	v.heartbeat()
	go func() {
		for {
			select {
			case <-ticker.C:
				v.heartbeat()
			case <-v.close:
				ticker.Stop()
				return
			}
		}
	}()
}

func (v *VoiceConnection) heartbeat() {
	v.wsMutex.Lock()
	err := v.ws.WriteJSON(voiceHeartbeatOp{3, int(time.Now().Unix())})
	v.wsMutex.Unlock()
	if err != nil {
		log.Printf("error sending heartbeat to voice endpoint %s, %s", v.Endpoint, err)

		ce, ok := err.(*websocket.CloseError)
		if ok {
			log.Printf("websocket failed: %v - %v", ce.Code, ce.Text)
			v.Close()
		}
	}
}

func (v *VoiceConnection) udpOpen() (udp *net.UDPConn, err error) {
	v.Mutex.Lock()
	defer v.Mutex.Unlock()

	if v.ws == nil {
		return nil, ErrVoiceWSNil
	}

	if v.udpclose == nil {
		return nil, ErrNilClose
	}

	if v.Endpoint == "" {
		return nil, ErrEmptyEndpoint
	}

	host := fmt.Sprintf("%v:%v", v.op2.IP, v.op2.Port)
	addr, err := net.ResolveUDPAddr("udp", host)
	if err != nil {
		return nil, err
	}

	udp, err = net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}

	send := make([]byte, 70)
	binary.BigEndian.PutUint32(send, v.op2.SSRC)
	_, err = udp.Write(send)
	if err != nil {
		return nil, err
	}

	recv := make([]byte, 70)
	recvLen, _, err := udp.ReadFromUDP(recv)
	if err != nil {
		return
	}

	if recvLen < 70 {
		return nil, ErrUDPSmallPacket
	}

	var ip string
	for i := 4; i < 20; i++ {
		if recv[i] == 0 {
			break
		}
		ip = string(recv[i])
	}

	port := binary.BigEndian.Uint16(recv[68:70])

	data := voiceUDPOp{1, voiceUDPD{"udp", voiceUDPData{ip, port, "xsalsa20_poly1305"}}}
	v.wsMutex.Lock()
	err = v.ws.WriteJSON(data)
	v.wsMutex.Unlock()

	udpclose := make(chan struct{})
	v.udpclose = udpclose

	go v.udpKeepAlive(udp, udpclose, 5*time.Second)
	return
}

func (v *VoiceConnection) udpKeepAlive(udp *net.UDPConn, udpclose <-chan struct{}, i time.Duration) {
	if udp == nil || udpclose == nil {
		return
	}

	var err error
	var seq uint64

	packet := make([]byte, 8)

	ticker := time.NewTicker(i)
	defer ticker.Stop()
	for {
		binary.LittleEndian.PutUint64(packet, seq)
		seq++

		_, err = udp.Write(packet)
		if err != nil {
			log.Printf("error: %v", err)
			return
		}

		select {
		case <-ticker.C:
			//continue
		case <-udpclose:
			return
		}
	}

}

func (v *VoiceConnection) SetSpeaking(speaking bool) (err error) {
	type voiceSpeakingData struct {
		Speaking bool `json:"speaking"`
		Delay    int  `json:"delay"`
	}

	type voiceSpeakingOp struct {
		Op   int               `json:"op"` // Always 5
		Data voiceSpeakingData `json:"d"`
	}
	v.wsMutex.Lock()
	if v.ws == nil {
		v.wsMutex.Unlock()
		return fmt.Errorf("no VoiceConnection websocket")
	}

	data := voiceSpeakingOp{5, voiceSpeakingData{speaking, 0}}

	err = v.ws.WriteJSON(data)
	v.wsMutex.Unlock()

	v.Mutex.Lock()
	defer v.Mutex.Unlock()
	if err != nil {
		v.Speaking = false
		log.Fatalln(err)
		return
	}

	v.Speaking = speaking

	return
}
