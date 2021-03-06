package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/damaredayo/gopherlink"
	pb "github.com/damaredayo/gopherlink/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var players = make(map[string]*gopherlink.VoiceConnection, 0)

type rpc struct {
	pb.GopherlinkServer
	Ok int64
}

func (r *rpc) GetStatusStream(_ *emptypb.Empty, stream pb.Gopherlink_GetStatusStreamServer) error {

	for {
		for k, v := range players {
			stream.Send(&pb.Status{
				Ok:      v.Ready,
				GuildId: k,
				Playing: pb.PlayStatus_STOPPED,
				Usage: &pb.Usage{
					Ram: rand.Float32() * 100,
					Cpu: rand.Float32() * 100,
				},
			})
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (r *rpc) AddSong(ctx context.Context, song *pb.SongRequest) (*pb.SongAdded, error) {
	var songInfo *pb.SongInfo

	v, ok := players[song.GuildId]
	if !ok {
		return nil, fmt.Errorf("no player to guildid")
	}

	if v.NowPlaying != nil {
		info, err := gopherlink.YoutubeToInfo(song.GetURL())
		if err != nil {
			return nil, err
		}
		songInfo = &pb.SongInfo{
			GuildId:  song.GetGuildId(),
			Playing:  pb.PlayStatus_STOPPED,
			Duration: int64(info.Duration),
			Elapsed:  0,
			Author:   info.Uploader,
			Title:    info.Title,
			URL:      song.GetURL(),
		}
		v.Queue.AddSong(ctx, songInfo)
		v.Cache.PreloadSong(song.GetURL())
	} else {
		aac, info, err := gopherlink.YoutubeToAAC(song.GetURL())
		if err != nil {
			return nil, err
		}

		v.NowPlaying = &gopherlink.Np{
			GuildId:  song.GuildId,
			Playing:  true,
			Duration: int64(info.Duration),
			Elapsed:  v.GetElapsed(),
			Author:   info.Uploader,
			Title:    info.Title,
		}

		songInfo = &pb.SongInfo{
			GuildId:  song.GetGuildId(),
			Playing:  pb.PlayStatus_PLAYING,
			Duration: v.NowPlaying.Duration,
			Elapsed:  0,
			Author:   info.Uploader,
			Title:    info.Title,
			URL:      song.GetURL(),
		}

		if !v.Reconnecting {
			pcm, rate := gopherlink.AacToPCM(aac)
			v.SetPCM(pcm)
			v.Volume = 1.0
			go v.MusicPlayer(rate, 960)
		}
	}

	sa := &pb.SongAdded{
		Song: song,
		Info: songInfo,
	}

	return sa, nil
}

func (r *rpc) RemoveSong(ctx context.Context, song *pb.SongInfo) (*pb.SongRemoved, error) {
	sr := &pb.SongRemoved{
		Song: song,
		Ok:   false,
	}
	return sr, nil
}

func (r *rpc) PauseSong(ctx context.Context, req *pb.SongPauseRequest) (*pb.SongInfo, error) {
	guildId := req.GetGuildId()
	player, ok := players[guildId]
	if !ok {
		return nil, fmt.Errorf("no player to guildid")
	}

	if player.NowPlaying == nil {
		return nil, fmt.Errorf("nothing playing")
	}

	player.Paused = !player.Paused
	playing := pb.PlayStatus_PAUSED
	ss := &pb.SongInfo{
		GuildId:  guildId,
		Playing:  playing,
		Duration: player.NowPlaying.Duration,
		Elapsed:  player.GetElapsed(),
		Author:   player.NowPlaying.Author,
		Title:    player.NowPlaying.Title,
	}
	return ss, nil
}

func (r *rpc) StopSong(ctx context.Context, req *pb.SongStopRequest) (*pb.SongInfo, error) {
	guildId := req.GetGuildId()
	player, ok := players[guildId]
	if !ok {
		return nil, fmt.Errorf("no player to guildid")
	}

	if player.NowPlaying == nil {
		return nil, fmt.Errorf("nothing playing")
	}

	player.Stop()
	playing := pb.PlayStatus_STOPPED
	ss := &pb.SongInfo{
		GuildId:  guildId,
		Playing:  playing,
		Duration: player.NowPlaying.Duration,
		Elapsed:  player.GetElapsed(),
		Author:   player.NowPlaying.Author,
		Title:    player.NowPlaying.Title,
	}
	return ss, nil
}

func (r *rpc) NowPlaying(ctx context.Context, np *pb.NowPlayingRequest) (*pb.SongInfo, error) {
	guildId := np.GetGuildId()
	player, ok := players[guildId]
	if !ok {
		player = &gopherlink.VoiceConnection{Playing: false}
	}

	if player.NowPlaying == nil {
		return nil, fmt.Errorf("nothing playing")
	}

	playing := pb.PlayStatus_STOPPED
	if player.Playing {
		playing = pb.PlayStatus_PLAYING
	}
	ss := &pb.SongInfo{
		GuildId:  guildId,
		Playing:  playing,
		Duration: player.NowPlaying.Duration,
		Elapsed:  player.GetElapsed(),
		Author:   player.NowPlaying.Author,
		Title:    player.NowPlaying.Title,
	}
	return ss, nil
}

func (r *rpc) GetQueue(ctx context.Context, req *pb.QueueRequest) (*pb.Queue, error) {
	guildId := req.GetGuildId()
	player, ok := players[guildId]
	if !ok {
		return nil, fmt.Errorf("no player to guildid")
	}
	queue, err := player.Queue.GetQueue(ctx)
	if err != nil {
		return nil, err
	}

	q := &pb.Queue{
		GuildId: guildId,
		Songs:   queue,
	}
	return q, nil
}

func (r *rpc) Seek(ctx context.Context, seek *pb.SeekRequest) (*pb.SongInfo, error) {
	guildId := seek.GetGuildId()
	player, ok := players[guildId]
	if !ok {
		return nil, fmt.Errorf("no player to guildid")
	}
	if player.NowPlaying == nil {
		return nil, fmt.Errorf("nothing playing")
	}
	newByteTrack := int(seek.GetDuration()) * 96000
	if newByteTrack > player.GetPCMLength() {
		return nil, fmt.Errorf("seek too far")
	}
	player.ByteTrack = newByteTrack
	ss := &pb.SongInfo{
		GuildId:  guildId,
		Playing:  pb.PlayStatus_PLAYING,
		Duration: player.NowPlaying.Duration,
		Elapsed:  player.GetElapsed(),
	}
	return ss, nil
}

func (r *rpc) Volume(ctx context.Context, vol *pb.VolumeRequest) (*pb.VolumeResponse, error) {
	guildId := vol.GetGuildId()
	player, ok := players[guildId]
	if !ok {
		return nil, fmt.Errorf("no player to guildid")
	}
	player.Volume = vol.GetVolume()
	return &pb.VolumeResponse{Ok: true, Volume: player.Volume}, nil
}

func (r *rpc) Loop(ctx context.Context, loop *pb.LoopRequest) (*pb.LoopResponse, error) {
	guildId := loop.GetGuildId()
	player, ok := players[guildId]
	if !ok {
		return nil, fmt.Errorf("no player to guildid")
	}
	player.Loop = loop.GetLoop()
	return &pb.LoopResponse{Ok: true, Loop: player.Loop}, nil
}

func (r *rpc) Skip(ctx context.Context, skip *pb.SkipRequest) (*pb.SongRemoved, error) {
	guildId := skip.GetGuildId()
	player, ok := players[guildId]
	if !ok {
		return nil, fmt.Errorf("no player to guildid")
	}
	if player.NowPlaying == nil {
		return nil, fmt.Errorf("nothing playing")
	}
	song := &pb.SongRemoved{
		Song: &pb.SongInfo{
			GuildId:  guildId,
			Playing:  pb.PlayStatus_PLAYING,
			Duration: player.NowPlaying.Duration,
			Author:   player.NowPlaying.Author,
			Title:    player.NowPlaying.Title,
		},
	}
	err := player.Skip()
	if err != nil {
		return nil, err
	}
	return song, nil
}

func (r *rpc) CreatePlayer(ctx context.Context, voiceData *pb.DiscordVoiceServer) (*pb.PlayerResponse, error) {
	pr := &pb.PlayerResponse{
		Ok: false,
		Player: &pb.Player{
			GuildId: voiceData.GuildId,
			Playing: pb.PlayStatus_STOPPED,
		},
	}

	vc := gopherlink.VoiceConnection{
		Token:     voiceData.Token,
		Endpoint:  voiceData.Endpoint,
		GuildID:   voiceData.GuildId,
		SessionID: voiceData.SessionId,
		UserID:    voiceData.UserId,
	}

	err := vc.MakeQueue(gopherlink.InternalQueueType, nil)
	if err != nil {
		return pr, err
	}

	err = vc.MakeCache(gopherlink.CacheTypeInternal, nil)
	if err != nil {
		return pr, err
	}

	err = vc.Open()
	if err != nil {
		return pr, err
	}

	pr.Ok = vc.Ready

	players[voiceData.GuildId] = &vc

	return pr, nil
}

func initRPC() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", 50051))
	if err != nil {
		log.Fatalf("gopherlink failed to open tcp listener: %v\n", err)
	}

	s := grpc.NewServer()

	pb.RegisterGopherlinkServer(s, &rpc{})
	log.Printf("Gopherlink RPC started on %v", listener.Addr())

	if err := s.Serve(listener); err != nil && err != grpc.ErrServerStopped {
		log.Fatalf("rpc failed to serve: %v", err)
	}
}
