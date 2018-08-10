package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

// Configuration filled from configuration file
type Configuration struct {
	SpotifyClientID  string
	SpotifySecretKey string
	YoutubeKey       string
	DiscordBotKey    string
}

var (
	configuration *Configuration
	maxResults    = flag.Int64("max-results", 1, "Max YouTube results")
	service       *youtube.Service
)

func main() {
	conf, err := loadConfiguration("config.json")
	if err != nil {
		log.Fatal("Can't load config file:", err)
	}
	configuration = &conf

	flag.Parse()

	initYoutube()

	// Discord
	dg, err := discordgo.New("Bot " + configuration.DiscordBotKey)
	if err != nil {
		log.Fatalf("error creating Discord session: %v", err)
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!music") {
		name := strings.TrimPrefix(m.Content, "!music")
		log.Printf("Request: %v", name)

		// Youtube
		urlYoutube, err := callYoutube(name)
		log.Printf("Response: [Youtube] %v", urlYoutube)
		handleError(err, "")

		urlSpotify := callSpotify(name)
		log.Printf("Response: [Spotify] %v", urlSpotify)

		url := "[Youtube] " + urlYoutube + "\n[Spotify] " + urlSpotify

		// Send
		s.ChannelMessageSend(m.ChannelID, url)
	} else if strings.HasPrefix(m.Content, "!list") { // Easter egg
		s.ChannelMessageSend(m.ChannelID, "ALLEZ, QUOI !")
	}
}

func initYoutube() {
	log.Printf("Youtube - Initializing Connection")
	client := &http.Client{
		Transport: &transport.APIKey{Key: configuration.YoutubeKey},
	}

	serviceRetrived, err := youtube.New(client)
	if err != nil {
		log.Fatalf("Error creating new YouTube client: %v", err)
	}
	service = serviceRetrived
}

func callYoutube(name string) (string, error) {
	if service == nil {
		initYoutube()
	}

	// Make the API call to YouTube.
	call := service.Search.List("id,snippet").
		Q(name).
		MaxResults(*maxResults).
		Type("video")
	response, err := call.Do()
	return extractURL(response), err
}

func callSpotify(name string) string {
	config := &clientcredentials.Config{
		ClientID:     configuration.SpotifyClientID,
		ClientSecret: configuration.SpotifySecretKey,
		TokenURL:     spotify.TokenURL,
	}
	token, err := config.Token(context.Background())
	if err != nil {
		log.Fatalf("couldn't get token: %v", err)
	}

	client := spotify.Authenticator{}.NewClient(token)

	results, err := client.Search(name, spotify.SearchTypeTrack)
	if err != nil {
		log.Fatal(err)
	}

	if results.Tracks != nil && results.Tracks.Tracks != nil && len(results.Tracks.Tracks) > 0 && results.Tracks.Tracks[0].ExternalURLs != nil && results.Tracks.Tracks[0].ExternalURLs["spotify"] != "" {
		return results.Tracks.Tracks[0].ExternalURLs["spotify"]
	}
	return "No content found"
}

func extractURL(response *youtube.SearchListResponse) string {
	if len(response.Items) == 0 {
		return "No content found"
	}
	return "https://www.youtube.com/watch?v=" + response.Items[0].Id.VideoId
}

func handleError(err error, message string) {
	if message == "" {
		message = "Error making API call"
	}
	if err != nil {
		log.Fatalf(message+": %v", err.Error())
	}
}

// loadConfiguration loads configuration from json file
func loadConfiguration(configurationFile string) (Configuration, error) {
	configuration := Configuration{}

	file, err := os.Open(configurationFile)
	if err != nil {
		return configuration, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&configuration)
	return configuration, err
}
