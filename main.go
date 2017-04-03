// Package main starts a Sibyl server
package main

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"

	rice "github.com/GeertJohan/go.rice"
	"github.com/gorilla/handlers"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/synacor/sibyl/server"
)

const defaultPort = 5000

var maxPort = int(math.Pow(2, 16) - 1)
var s *server.Server

func main() {
	// reminder, that viper will only look at the first config file it sees
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/sibyl")
	viper.SetEnvPrefix("sibyl")
	viper.BindEnv("port")
	viper.BindEnv("log_level")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("port", defaultPort)
	if err := viper.ReadInConfig(); err != nil {
		// viper requires a config file to be present for some reason. this will check for that error
		// and silently ignore it
		if _, isConfigFileNotFoundError := err.(viper.ConfigFileNotFoundError); !isConfigFileNotFoundError {
			panic(err)
		}
	}
	configureLogger()

	tbox := rice.MustFindBox("templates")
	sbox := rice.MustFindBox("static")

	s = server.New(tbox, sbox)
	mux := s.ServeMux()

	done := make(chan bool, 1)
	go serve(mux)
	go s.ListenForEvents(done)

	<-done
}

func configureLogger() {
	levelStr := viper.GetString("log_level")
	level, err := log.ParseLevel(levelStr)
	if err != nil {
		log.Fatalf("level %s does not exist", level)
	}
	log.SetLevel(level)
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
}

func serve(mux *http.ServeMux) {
	port := viper.GetInt("port")
	tlsPort := viper.GetInt("tls_port")
	forceTLS := viper.GetBool("force_tls")
	tlsPrivateKeyFile := viper.GetString("tls_private_key")
	tlsPublicKeyFile := viper.GetString("tls_public_key")

	if port <= 0 || port > maxPort {
		log.Fatalf("PORT must be 0 < PORT <= %d", maxPort)
	} else if tlsPort > 0 && port == tlsPort {
		log.Fatalf("PORT cannot equal TLS_PORT")
	} else if tlsPort > maxPort {
		log.Fatalf("TLS_PORT must be 0 < TLS_PORT <= %d", maxPort)
	} else if tlsPort > 0 && (tlsPublicKeyFile == "" || tlsPrivateKeyFile == "") {
		log.Fatal("must supply TLS_PRIVATE_KEY and TLS_PUBLIC_KEY if TLS_PORT specified")
	}

	if tlsPort > 0 {
		go func() {
			pstr := fmt.Sprintf(":%d", tlsPort)
			log.WithFields(log.Fields{"pid": os.Getpid()}).Printf("Listening on %s", pstr)

			log.Fatal(http.ListenAndServeTLS(pstr, tlsPublicKeyFile, tlsPrivateKeyFile, mux))
		}()
	}

	pstr := fmt.Sprintf(":%d", port)
	log.WithFields(log.Fields{"pid": os.Getpid()}).Printf("Listening on %s", pstr)
	log.Fatal(http.ListenAndServe(pstr, handlers.CombinedLoggingHandler(os.Stdout, maybeRedirectToTLS(tlsPort, forceTLS, mux))))
}

// maybeRedirectToTLS is middleware for optionally redirecting the user to the TLS version based on arguments passed to the application.
func maybeRedirectToTLS(tlsPort int, forceTLS bool, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if forceTLS && tlsPort > 0 {
			hostname := strings.Split(r.Host, ":")[0]
			if tlsPort != 443 {
				hostname += fmt.Sprintf(":%d", tlsPort)
			}

			url := "https://" + hostname + r.URL.String()
			http.Redirect(w, r, url, http.StatusMovedPermanently)
			return
		}

		h.ServeHTTP(w, r)
	})
}
