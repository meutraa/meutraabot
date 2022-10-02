package main

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/hostrouter"
	"golang.org/x/crypto/acme/autocert"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/meutraa/meutraabot/pkg/db"
	"github.com/nicklaw5/helix/v2"
	"github.com/pkg/errors"
)

func (s *Server) PrepareAPI() {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	ar := chi.NewRouter()
	ar.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"https://meuua.com"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: false,
		MaxAge:           3600,
	}))

	ar.Route("/channels", func(r chi.Router) {
		r.Get("/", s.listChannels())
		r.Route("/{id}", func(r chi.Router) {
			r.Put("/", s.registerChannel())
			r.Get("/", s.getChannel())
			r.Delete("/", s.unregisterChannel())
			r.Patch("/", s.patchChannel())
			r.Get("/commands", s.listCommands())
			r.Get("/approvals", s.listApprovals())
		})
	})

	ar.Route("/commands", func(r chi.Router) {
		r.Get("/", s.listLocalCommands("0"))
	})

	or := chi.NewRouter()
	or.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		// Get the "access_token" from the url fragment using javascript. Display the token large in the center of the page.
		w.Write([]byte(`<html>
				<head>
						<script>
								var url = window.location.href;
								var access_token = url.split("#")[1].split("&")[0].split("=")[1];
								document.write("<h1 style='text-align:center; font-size: 96px;'>" + access_token + "</h1>");
						</script>
				</head>
		</html>`))
	})

	hr := hostrouter.New()
	hr.Map("api.meuua.com", ar)
	hr.Map("oauth.meuua.com", or)

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("api.meuua.com", "oauth.meuua.com"),
		Cache:      autocert.DirCache("certs"),
	}

	r.Mount("/", hr)

	server := &http.Server{
		Addr:    ":https",
		Handler: r,
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}

	go func() {
		go http.ListenAndServe(":http", certManager.HTTPHandler(nil))
		err := server.ListenAndServeTLS("", "")
		if nil != err {
			panic(err)
		}
	}()
}

func (s *Server) listLocalCommands(channelID string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		commands, err := s.q.GetCommandsByID(r.Context(), channelID)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res, err := json.Marshal(commands)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(res)
	})
}

func (s *Server) registerChannel() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		idstr, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id = strconv.FormatInt(idstr, 10)

		token := r.Header.Get("Authorization")
		if len(token) == 0 {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		user, err := s.getUserFromToken(token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if user.ID != id {
			http.Error(w, "Not authorized to register this channel", http.StatusForbidden)
			return
		}

		if err := s.q.CreateChannel(r.Context(), user.ID); nil != err {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.JoinChannels([]string{user.Login}, []string{user.ID})
		msg := "Hi " + user.DisplayName + " 👋"
		go func() {
			time.Sleep(time.Second * 2)
			s.irc.Say(user.Login, msg)
		}()

		w.WriteHeader(http.StatusOK)
	})
}

func (s *Server) unregisterChannel() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		idstr, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id = strconv.FormatInt(idstr, 10)

		token := r.Header.Get("Authorization")
		if len(token) == 0 {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		user, err := s.getUserFromToken(token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if user.ID != id {
			http.Error(w, "Not authorized to unregister this channel", http.StatusForbidden)
			return
		}

		if err := s.q.DeleteChannel(r.Context(), user.ID); nil != err {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		go func() {
			time.Sleep(2 * time.Second)
			s.irc.Depart(user.Login)
		}()

		s.irc.Say(user.Login, "Bye "+user.DisplayName+"👋")

		w.WriteHeader(http.StatusOK)
	})
}

func (s *Server) listCommands() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		// verify id is an int
		idstr, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id = strconv.FormatInt(idstr, 10)

		commands, err := s.q.GetCommandsByID(r.Context(), id)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res, err := json.Marshal(commands)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(res)
	})
}

func (s *Server) listApprovals() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		// verify id is an int
		idstr, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id = strconv.FormatInt(idstr, 10)

		approvals, err := s.q.GetApprovals(r.Context(), id)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		channelIDs := []string{}
		for _, a := range approvals {
			channelIDs = append(channelIDs, a.UserID)
		}

		resp, err := Users(s.twitch, channelIDs, []string{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(res)
	})
}

func (s *Server) getUserFromToken(token string) (helix.User, error) {
	client, err := helix.NewClient(&helix.Options{
		ClientID:     s.env.twitchClientID,
		ClientSecret: s.env.twitchClientSecret,
	})
	if err != nil {
		return helix.User{}, errors.Wrap(err, "Unable to create twitch api client")
	}

	client.SetUserAccessToken(token)

	resp, err := client.GetUsers(&helix.UsersParams{})
	if err != nil {
		fmt.Println("Unable to get user", err)
		return helix.User{}, err
	}

	if len(resp.Data.Users) == 0 {
		return helix.User{}, errors.New("No user found")
	}

	return resp.Data.Users[0], nil
}

func (s *Server) patchChannel() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		token := r.Header.Get("Authorization")
		if len(token) == 0 {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		user, err := s.getUserFromToken(token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if user.ID != id && user.ID != s.env.twitchOwnerID {
			http.Error(w, "Not authorized to patch", http.StatusForbidden)
			return
		}

		// verify id is an int
		idstr, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id = strconv.FormatInt(idstr, 10)

		// parse Channel from request body
		var channel db.Channel
		err = json.NewDecoder(r.Body).Decode(&channel)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if channel.ReplySafety < 0 || channel.ReplySafety > 3 {
			http.Error(w, "reply safety must be between 0 and 3", http.StatusBadRequest)
			return
		}

		if channel.AutoreplyFrequency < 1 || channel.AutoreplyFrequency > 5 {
			http.Error(w, "autoreply frequency must be between 1 and 5", http.StatusBadRequest)
			return
		}

		if channel.OpenaiToken.String != "" && len(channel.OpenaiToken.String) > 8 {
			// check string matches regex "^sk-\w{48}$"
			regex, err := regexp.Compile("^sk-\\w{48}$")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if !regex.MatchString(channel.OpenaiToken.String) {
				http.Error(w, "openai token invalid format", http.StatusBadRequest)
				return
			}

			channel.OpenaiToken.Valid = true

			err = s.q.UpdateChannelToken(r.Context(), db.UpdateChannelTokenParams{
				ChannelID:   id,
				OpenaiToken: channel.OpenaiToken,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		err = s.q.UpdateChannel(r.Context(), db.UpdateChannelParams{
			ChannelID:          id,
			AutoreplyEnabled:   channel.AutoreplyEnabled,
			AutoreplyFrequency: channel.AutoreplyFrequency,
			ReplySafety:        channel.ReplySafety,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.getChannel()(w, r)
	})
}

func (s *Server) getChannel() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		// verify id is an int
		idstr, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id = strconv.FormatInt(idstr, 10)

		channel, err := s.q.GetChannel(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if channel.OpenaiToken.Valid {
			channel.OpenaiToken.String = "******"
		}

		res, err := json.Marshal(channel)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(res)
	})
}

func (s *Server) listChannels() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		channels, err := s.q.GetChannels(r.Context())
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp, err := Users(s.twitch, channels, []string{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(res)
	})
}
