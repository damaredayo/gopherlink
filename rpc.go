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
	aac, info, err := youtubeToAAC(song.GetURL())
	if err != nil {
		return nil, err
	}
	log.Printf("downloaded %v successfully.", info.Title)

	v, ok := players[song.GuildId]
	if !ok {
		return nil, fmt.Errorf("no player to guildid")
	}
	v.NowPlaying = &np{
		GuildId:  song.GuildId,
		Playing:  true,
		Duration: int64(info.Duration),
		Started:  time.Now(),
		Author:   info.Uploader,
		Title:    info.Title,
	}

	if !v.Reconnecting {
		pcm, rate := aacToPCM(aac)
		v.pcm = pcm
		playerclose := make(chan struct{})
		go v.musicPlayer(playerclose, rate, 960)
	}

	sa := &pb.SongAdded{
		Song: song,
		Info: &pb.SongInfo{
			GuildId:  song.GetGuildId(),
			Playing:  pb.PlayStatus_PLAYING,
			Duration: v.NowPlaying.Duration,
			Elapsed:  0,
			Author:   info.Uploader,
			Title:    info.Title,
		},
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

func (r *rpc) Seek(ctx context.Context, seek *pb.SeekRequest) (*pb.SongInfo, error) {
	ss := &pb.SongInfo{
		GuildId:  "none",
		Playing:  pb.PlayStatus_PAUSED,
		Duration: 1738,
		Elapsed:  727,
	}
	return ss, nil
}

func (r *rpc) CreatePlayer(ctx context.Context, voiceData *pb.DiscordVoiceServer) (*pb.PlayerResponse, error) {
	fmt.Println(voiceData)
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
	err := vc.Open()
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
