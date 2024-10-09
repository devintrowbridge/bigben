package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
)

var token string
var guildId string
var buffer = make([][]byte, 0)

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.StringVar(&guildId, "g", "", "Guild ID")
	flag.Parse()
}

func main() {
	now := time.Now()
	fmt.Println("Running bigben for ", now.String())

	if token == "" {
		fmt.Println("No token provided. Please run: airhorn -t <bot token>")
		return
	}

	// Load the sound file.
	err := loadSound()
	if err != nil {
		fmt.Println("Error loading sound: ", err)
		fmt.Println("Please copy bigben.dca to this directory.")
		return
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildVoiceStates | discordgo.IntentGuildMembers

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
		return
	}

	channel := findPopulatedVoice(dg)
	if channel == nil {
		fmt.Println("No users in any channels")
		return
	}
	fmt.Println("Joining channel", channel.Name)

	err = playSound(dg, channel.GuildID, channel.ID)
	if err != nil {
		fmt.Println("Error playing sound:", err)
	}

	// Cleanly close down the Discord session.
	dg.Close()
}

func findPopulatedVoice(s *discordgo.Session) *discordgo.Channel {
	channels, err := s.GuildChannels(guildId)
	if err != nil {
		fmt.Println("Error fetching channels: ", err)
		return nil
	}
	// Get all the guild members
	members, err := s.GuildMembers(guildId, "", 1000) // Fetching up to 1000 members at once
	if err != nil {
		fmt.Println("Error fetching members: ", err)
		return nil
	}

	// Create a map to track how many users are in each voice channel
	voiceChannelCounts := make(map[string]int)

	// Loop through each member and check if they are in a voice channel
	for _, member := range members {
		vs, _ := s.State.VoiceState(guildId, member.User.ID)
		if vs != nil && vs.ChannelID != "" {
			voiceChannelCounts[vs.ChannelID]++
		}
	}

	// Find the first voice channel that has at least one user
	for _, channel := range channels {
		if channel.Type == discordgo.ChannelTypeGuildVoice {
			if voiceChannelCounts[channel.ID] > 0 {
				return channel
			}
		}
	}

	// If no populated channel is found, return nil
	return nil
}

// loadSound attempts to load an encoded sound file from disk.
func loadSound() error {
	file, err := os.Open("bigben.dca")
	if err != nil {
		fmt.Println("Error opening dca file :", err)
		return err
	}

	var opuslen int16

	for {
		// Read opus frame length from dca file.
		err = binary.Read(file, binary.LittleEndian, &opuslen)

		// If this is the end of the file, just return.
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			err := file.Close()
			if err != nil {
				return err
			}
			return nil
		}

		if err != nil {
			fmt.Println("Error reading from dca file :", err)
			return err
		}

		// Read encoded pcm from dca file.
		InBuf := make([]byte, opuslen)
		err = binary.Read(file, binary.LittleEndian, &InBuf)

		// Should not be any end of file errors
		if err != nil {
			fmt.Println("Error reading from dca file :", err)
			return err
		}

		// Append encoded pcm data to the buffer.
		buffer = append(buffer, InBuf)
	}
}

// playSound plays the current buffer to the provided channel.
func playSound(s *discordgo.Session, guildID string, channelID string) (err error) {

	// Join the provided voice channel.
	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return err
	}

	// Sleep for a specified amount of time before playing the sound
	time.Sleep(250 * time.Millisecond)

	// Start speaking.
	vc.Speaking(true)

	currentTime := time.Now()
	currentHour := currentTime.Hour() % 12
	if currentHour == 0 {
		currentHour = 12
	}

	// Send the buffer data.
	start := time.Now()

	// chime a number of times equal to the hour of the day
	for chime := 0; chime < currentHour; chime++ {

		// play the chime sound
		for _, buff := range buffer {
			vc.OpusSend <- buff
			elapsed := time.Since(start)

			// space chimes out 4 seconds apart
			if elapsed.Milliseconds() > 4000 && chime != currentHour-1 {
				start = time.Now()
				break
			}
		}
	}

	// Stop speaking
	vc.Speaking(false)

	// Sleep for a specificed amount of time before ending.
	time.Sleep(250 * time.Millisecond)

	// Disconnect from the provided voice channel.
	vc.Disconnect()

	return nil
}
