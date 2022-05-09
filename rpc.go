package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	pb "github.com/damaredayo/gopherlink/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type rpc struct {
	pb.GopherlinkServer
	Ok int64
}

func (r *rpc) GetStatusStream(_ *emptypb.Empty, stream pb.Gopherlink_GetStatusStreamServer) error {

	for {
		stream.Send(&pb.Status{
			Ok:      r.Ok,
			Playing: pb.PlayStatus_STOPPED,
			Usage: &pb.Usage{
				Ram: rand.Float32() * 100,
				Cpu: rand.Float32() * 100,
			},
		})
		time.Sleep(100 * time.Millisecond)
	}
}

func (r *rpc) AddSong(ctx context.Context, song *pb.SongRequest) (*pb.SongAdded, error) {
	var songInfo *pb.SongInfo

	v, ok := players[song.GuildId]
	if !ok {
		return nil, fmt.Errorf("no player to guildid")
	}

	if v.Playing || v.paused {
		info, err := youtubeToInfo(song.GetURL())
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
	} else {
		aac, info, err := youtubeToAAC(song.GetURL())
		if err != nil {
			return nil, err
		}

		v.NowPlaying = &np{
			GuildId:  song.GuildId,
			Playing:  true,
			Duration: int64(info.Duration),
			Started:  time.Now(),
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
			pcm, rate := aacToPCM(aac)
			v.pcm = pcm
			v.playerclose = make(chan struct{})
			go v.musicPlayer(rate, 960)
		}
	}

	sa := &pb.SongAdded{
		Song: song,
		Info: songInfo,
	}

	return sa, nil
}

func (r *rpc) RemoveSong(ctx context.Context, song *pb.SongRequest) (*pb.SongRemoved, error) {
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

	player.paused = !player.paused
	playing := pb.PlayStatus_PAUSED
	ss := &pb.SongInfo{
		GuildId:  guildId,
		Playing:  playing,
		Duration: player.NowPlaying.Duration,
		Elapsed:  int64(time.Since(player.NowPlaying.Started).Seconds()),
		Author:   player.NowPlaying.Author,
		Title:    player.NowPlaying.Title,
	}
	return ss, nil
}

func (r *rpc) NowPlaying(ctx context.Context, np *pb.NowPlayingRequest) (*pb.SongInfo, error) {
	guildId := np.GetGuildId()
	player, ok := players[guildId]
	if !ok {
		player = &VoiceConnection{Playing: false}
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
		Elapsed:  int64(time.Since(player.NowPlaying.Started).Seconds()),
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
	player.ByteTrack = int(seek.GetDuration()) * 96000
	ss := &pb.SongInfo{
		GuildId:  guildId,
		Playing:  pb.PlayStatus_PLAYING,
		Duration: player.NowPlaying.Duration,
		Elapsed:  int64(time.Since(player.NowPlaying.Started).Seconds()),
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

func (r *rpc) CreatePlayer(ctx context.Context, voiceData *pb.DiscordVoiceServer) (*pb.PlayerResponse, error) {
	pr := &pb.PlayerResponse{
		Ok: false,
		Player: &pb.Player{
			GuildId: voiceData.GuildId,
			Playing: pb.PlayStatus_STOPPED,
		},
	}

	vc := VoiceConnection{
		Token:     voiceData.Token,
		Endpoint:  voiceData.Endpoint,
		GuildID:   voiceData.GuildId,
		SessionID: voiceData.SessionId,
		UserID:    voiceData.UserId,
	}

	err := vc.MakeQueue(InternalQueueType, nil)
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
