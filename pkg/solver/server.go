package solver

import (
	"context"
	"encoding/json"
	"fmt"
	corehttp "net/http"
	"time"

	"github.com/bacalhau-project/lilypad/pkg/data"
	"github.com/bacalhau-project/lilypad/pkg/http"
	"github.com/bacalhau-project/lilypad/pkg/solver/store"
	"github.com/bacalhau-project/lilypad/pkg/system"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type solverServer struct {
	options    http.ServerOptions
	controller *SolverController
	store      store.SolverStore
}

func NewSolverServer(
	options http.ServerOptions,
	controller *SolverController,
	store store.SolverStore,
) (*solverServer, error) {
	server := &solverServer{
		options:    options,
		controller: controller,
		store:      store,
	}
	return server, nil
}

func (solverServer *solverServer) ListenAndServe(ctx context.Context, cm *system.CleanupManager) error {
	router := mux.NewRouter()

	subrouter := router.PathPrefix("/api/v1").Subrouter()

	subrouter.Use(http.CorsMiddleware)

	subrouter.HandleFunc("/job_offers", http.Wrapper(solverServer.getJobOffers)).Methods("GET")
	subrouter.HandleFunc("/job_offers", http.Wrapper(solverServer.addJobOffer)).Methods("POST")

	subrouter.HandleFunc("/resource_offers", http.Wrapper(solverServer.getResourceOffers)).Methods("GET")
	subrouter.HandleFunc("/resource_offers", http.Wrapper(solverServer.addResourceOffer)).Methods("POST")

	// this will fan out to all connected web socket connections
	// we read all events coming from inside the solver controller
	// and write them to anyone who is connected to us
	websocketEventChannel := make(chan []byte)

	// listen to events coming out of the controller and write them to all
	// connected websocket connections
	go func() {
		select {
		case ev := <-solverServer.controller.getEventChannel():
			// we write this event to all connected web socket connections
			fmt.Printf("got controller event: %+v\n", ev)
			evBytes, err := json.Marshal(ev)
			if err != nil {
				log.Error().Msgf("Error marshalling event: %s", err.Error())
			}
			websocketEventChannel <- evBytes
			return
		case <-ctx.Done():
			return
		}
	}()

	http.StartWebSocketServer(
		subrouter,
		"/ws",
		websocketEventChannel,
		ctx,
	)

	srv := &corehttp.Server{
		Addr:              fmt.Sprintf("%s:%d", solverServer.options.Host, solverServer.options.Port),
		WriteTimeout:      time.Minute * 15,
		ReadTimeout:       time.Minute * 15,
		ReadHeaderTimeout: time.Minute * 15,
		IdleTimeout:       time.Minute * 60,
		Handler:           router,
	}

	// Create a channel to receive errors from ListenAndServe
	serverErrors := make(chan error, 1)

	// Run ListenAndServe in a goroutine because it blocks
	go func() {
		serverErrors <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		// Create a context with a timeout for the server to close
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Attempt to gracefully shut down the server
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to stop server: %w", err)
		}
	}

	return nil
}

func (solverServer *solverServer) getJobOffers(res corehttp.ResponseWriter, req *corehttp.Request) ([]data.JobOffer, error) {
	query := store.GetJobOffersQuery{}
	// if there is a job_creator query param then assign it
	if jobCreator := req.URL.Query().Get("job_creator"); jobCreator != "" {
		query.JobCreator = jobCreator
	}
	return solverServer.store.GetJobOffers(query)
}

func (solverServer *solverServer) getResourceOffers(res corehttp.ResponseWriter, req *corehttp.Request) ([]data.ResourceOffer, error) {
	query := store.GetResourceOffersQuery{}
	// if there is a job_creator query param then assign it
	if resourceProvider := req.URL.Query().Get("resource_provider"); resourceProvider != "" {
		query.ResourceProvider = resourceProvider
	}
	return solverServer.store.GetResourceOffers(query)
}

func (solverServer *solverServer) addJobOffer(res corehttp.ResponseWriter, req *corehttp.Request) (*data.JobOffer, error) {
	signerAddress, err := http.GetAddressFromHeaders(req)
	if err != nil {
		return nil, err
	}
	jobOffer, err := http.ReadBody[data.JobOffer](req)
	if err != nil {
		return nil, err
	}
	// only the job creator can post a job offer
	if signerAddress != jobOffer.JobCreator {
		return nil, fmt.Errorf("job creator address does not match signer address")
	}
	return solverServer.controller.addJobOffer(jobOffer)
}

func (solverServer *solverServer) addResourceOffer(res corehttp.ResponseWriter, req *corehttp.Request) (*data.ResourceOffer, error) {
	signerAddress, err := http.GetAddressFromHeaders(req)
	if err != nil {
		return nil, err
	}
	resourceOffer, err := http.ReadBody[data.ResourceOffer](req)
	if err != nil {
		return nil, err
	}
	// only the job creator can post a job offer
	if signerAddress != resourceOffer.ResourceProvider {
		return nil, fmt.Errorf("resource provider address does not match signer address")
	}
	return solverServer.controller.addResourceOffer(resourceOffer)
}
