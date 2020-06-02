package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/neelchoudhary/budgetwallet-api-server/services/plaidfinances"
	"github.com/neelchoudhary/budgetwallet-api-server/utils"
	"github.com/neelchoudhary/budgetwallet-webhook-service/controllers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	serverEnv := flag.String("serverEnv", "local", "Server Environment (local, prd)")
	serverWebhookPort := flag.String("serverWebhookPort", "8081", "Webhook Server Port")

	APIServerAddress := flag.String("apiServerAddress", "localhost:50051", "API Server Address")
	APIServerTLSCaPath := flag.String("apiServerTLSCaPath", "", "API Server TLS CA Path")
	APIServerAccessToken := flag.String("apiServerAccessToken", "", "JWT Access Token")
	flag.Parse()

	// Init router
	r := mux.NewRouter()

	apiServerConn, err := NewAPIServerConn(*APIServerAddress, *APIServerTLSCaPath, *APIServerAccessToken)
	if err != nil {
		log.Fatal(err)
	}
	defer apiServerConn.clientConn.Close()

	pf := plaidfinances.NewPlaidFinancesServiceClient(apiServerConn.clientConn)

	webhookController := controllers.NewWebhookController(pf)
	r.HandleFunc("/plaidwebhook/{user_id}", webhookController.ReceiveWebhook).Methods("POST")

	// Start server
	log.Println("Starting Webhook Server..." + *serverEnv)
	err = http.ListenAndServe(":"+*serverWebhookPort, r)
	if err != nil {
		log.Fatal(err)
	}
}

// APIServerConn ...
type APIServerConn struct {
	clientConn *grpc.ClientConn
}

// NewAPIServerConn creates new APIServerConn
func NewAPIServerConn(address string, tlsCaPath string, accessToken string) (*APIServerConn, error) {
	creds, err := credentials.NewClientTLSFromFile(tlsCaPath, "neelchoudhary.com")
	utils.LogIfFatalAndExit(err, "Error while loading CA trust certificate:")
	opts := grpc.WithTransportCredentials(creds)

	jwtCreds := utils.GetTokenAuth(accessToken)
	conn, err := grpc.Dial(address, opts, grpc.WithPerRPCCredentials(jwtCreds))
	if err != nil {
		return nil, err
	}

	return &APIServerConn{clientConn: conn}, nil
}
