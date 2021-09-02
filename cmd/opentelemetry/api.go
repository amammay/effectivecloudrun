package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"golang.org/x/sync/errgroup"
	"net/http"
	"time"
)

func (s *server) routes() {
	// setup otelmux middleware, this will auto create spans for processing within the mux realm
	// such as status code and other http attributes
	s.router.Use(otelmux.Middleware(AppName))
	apiRouter := s.router.PathPrefix("/api").Subrouter()

	func(r *mux.Router) {
		// we will focus on http related traces
		r.HandleFunc("/http", s.handleCallUpstreamHttpRequest()).Methods(http.MethodGet)

		r.HandleFunc("/grpc", s.handleCallUpstreamGrpcRequest()).Methods(http.MethodGet)

	}(apiRouter)
}

// handleCallUpstreamHttpRequest is our handler for http endpoint
func (s *server) handleCallUpstreamHttpRequest() http.HandlerFunc {
	client := s.bin
	return func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		// create our logger instance that is decorated with trace context
		logger := s.logger.WrapTraceContext(ctx)

		// do some sort of heavy processing
		val, err := client.doHeavyProcessingSerial(ctx)
		if err != nil {
			logger.Errorw("client.doHeavyProcessingSerial()", "err", err)
			s.respondJSON(writer, createErrorMessage(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		logger.Debug("finished doHeavyProcessingSerial()")

		// do more heavy processing
		val, err = client.doHeavyProcessingConcurrent(ctx)
		if err != nil {
			logger.Errorw("client.doHeavyProcessingConcurrent()", "err", err)
			s.respondJSON(writer, createErrorMessage(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		logger.Debug("finished doHeavyProcessingConcurrent()")

		s.respondJSON(writer, &val, http.StatusOK)
	}
}

func (s *server) handleCallUpstreamGrpcRequest() http.HandlerFunc {
	fs := s.firestore
	gofakeit.Seed(0)

	type beer struct {
		Created  time.Time `json:"created" firestore:"created,serverTimestamp"`
		BeerName string    `json:"beer_name" firestore:"beer_name"`
		DocID    string    `json:"doc_id" firestore:"doc_id"`
	}

	return func(writer http.ResponseWriter, request *http.Request) {
		ctx, span := startSpan(request.Context(), "server.handleCallUpstreamGrpcRequest()")
		defer span.End()
		// create our logger instance that is decorated with trace context
		logger := s.logger.WrapTraceContext(ctx)

		docRef := fs.Collection("beer").NewDoc()
		_, err := docRef.Create(ctx, &beer{
			BeerName: gofakeit.BeerName(),
			DocID:    docRef.ID,
		})
		if err != nil {
			logger.Errorw("fs.Collection(beer).Create()", "path", docRef.Path, "err", err)
			s.respondJSON(writer, createErrorMessage(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		today := time.Now().UTC().Truncate(24 * time.Hour)
		tomorrow := today.AddDate(0, 0, 1)
		all, err := fs.Collection("beer").
			Where("created", ">=", today).
			Where("created", "<", tomorrow).
			Documents(ctx).GetAll()
		if err != nil {
			logger.Errorw("fs.Collection(beer).Where", "created <", today, "path", docRef.Path, "err", err)
			s.respondJSON(writer, createErrorMessage(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		logger.Debugf("located %d beers created today", len(all))
		var beers []*beer
		for _, snapshot := range all {
			b := &beer{}
			err := snapshot.DataTo(b)
			if err != nil {
				logger.Errorw("snapshot.DataTo", "path", snapshot.Ref.Path, "err", err)
				s.respondJSON(writer, createErrorMessage(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			beers = append(beers, b)
		}

		s.respondJSON(writer, beers, http.StatusOK)
	}
}

// respondJSON is an util method for writing json back
func (s *server) respondJSON(writer http.ResponseWriter, data interface{}, statusCode int) {
	marshal, err := json.Marshal(data)
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)

	if _, err := writer.Write(marshal); err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

type errResponse struct {
	Message string `json:"message"`
}

func createErrorMessage(httpStatusCode int) *errResponse {
	return &errResponse{Message: http.StatusText(httpStatusCode)}
}

type binClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewBinClient(httpClient *http.Client, baseURL string) *binClient {
	if httpClient == nil {
		client := &http.Client{
			Timeout: 30 * time.Second,
		}
		httpClient = client
	}
	return &binClient{httpClient: httpClient, baseURL: baseURL}
}

type binJson struct {
	Slideshow struct {
		Author string `json:"author"`
		Date   string `json:"date"`
		Slides []struct {
			Title string   `json:"title"`
			Type  string   `json:"type"`
			Items []string `json:"items,omitempty"`
		} `json:"slides"`
		Title string `json:"title"`
	} `json:"slideshow"`
}

func (i *binClient) doHeavyProcessingSerial(ctx context.Context) (*binJson, error) {
	ctx, span := startSpan(ctx, "binClient.doHeavyProcessingSerial")
	defer span.End()

	m1 := make(map[string]interface{})
	if err := i.makeCall(ctx, "delay/6", http.MethodPost, &m1); err != nil {
		return nil, fmt.Errorf("i.makeCall(delay/6): %v", err)
	}

	b := &binJson{}
	if err := i.makeCall(ctx, "json", http.MethodGet, b); err != nil {
		return nil, fmt.Errorf("i.makeCall(json): %v", err)
	}
	return b, nil
}

func (i *binClient) doHeavyProcessingConcurrent(ctx context.Context) (*binJson, error) {
	ctx, span := startSpan(ctx, "binClient.doHeavyProcessingConcurrent")
	defer span.End()

	binChan := make(chan *binJson, 1)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		m1 := make(map[string]interface{})
		if err := i.makeCall(ctx, "delay/6", http.MethodPost, &m1); err != nil {
			return fmt.Errorf("i.makeCall(delay/6): %v", err)
		}
		return nil
	})

	g.Go(func() error {
		b := &binJson{}
		if err := i.makeCall(ctx, "json", http.MethodGet, b); err != nil {
			return fmt.Errorf("i.makeCall(json): %v", err)
		}
		binChan <- b
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("g.Wait(): %v", err)
	}
	b := <-binChan

	return b, nil
}

func (i *binClient) makeCall(ctx context.Context, url, method string, responseData interface{}) error {
	path := fmt.Sprintf("%s/%s", i.baseURL, url)
	req, err := http.NewRequestWithContext(ctx, method, path, nil)
	if err != nil {
		return fmt.Errorf("http.NewRequestWithContext(): %v", err)
	}

	do, err := i.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("i.httpClient.Do(): %v", err)
	}
	defer do.Body.Close()

	if do.StatusCode != http.StatusOK {
		return fmt.Errorf("bin non 200 status code: %d, status: %q", do.StatusCode, do.Status)
	}

	if err := json.NewDecoder(do.Body).Decode(responseData); err != nil {
		return fmt.Errorf("json.NewDecoder(): %v", err)
	}
	return nil
}
